#!/bin/bash

dryrun=false

while getopts ":h:c:k:t:p:r:d" opt; do
  case $opt in
    h)
      hostname=$OPTARG
      ;;
    c)
      if ! openssl x509 -noout -text -in $OPTARG &>/dev/null; then
        echo "ERROR: Provided file is not a valid certificate." >&2
        exit 1
      else
        cert_file=$OPTARG
      fi
      ;;
    k)
      if ! openssl rsa -noout -text -in $OPTARG &>/dev/null; then
        echo "ERROR: Provided file is not a valid private key." >&2
        exit 1
      else
        key_file=$OPTARG
      fi
      ;;
    t)
      token=$OPTARG
      ;;
    p)
      project=$OPTARG
      ;;
    r)
      route=$OPTARG
      ;;
    d)
      dryrun=true
      ;;
    \?)
      echo "ERROR: Invalid option -$OPTARG." >&2
      exit 1
      ;;
    :)
      echo "ERROR: Option -$OPTARG requires an argument." >&2
      exit 1
      ;;
  esac
done

shift $((OPTIND-1))

showsyntax() {
  echo "Syntax: $0 -t OPENSHIFT_TOKEN -h HOSTNAME -c CERTIFICATE_FILE -k KEY_FILE [-d] (dry-run)"
}

if [ -z $hostname ]; then
  echo "ERROR: Option -h is required." >&2
  showsyntax
  exit 1
fi
if [ -z $cert_file ]; then
  echo "ERROR: Option -c is required." >&2
  showsyntax
  exit 1
fi
if [ -z $key_file ]; then
  echo "ERROR: Option -k is required." >&2
  showsyntax
  exit 1
fi
if [ -z $token ]; then
  echo "ERROR: Option -t is required." >&2
  showsyntax
  exit 1
fi
if [ -z $project ]; then
  echo "ERROR: Option -p is required." >&2
  showsyntax
  exit 1
fi
if [ -z $route ]; then
  echo "ERROR: Option -r is required." >&2
  showsyntax
  exit 1
fi

OIFS="$IFS"
IFS=';'

#oc project openshift >/dev/null

# Get all the necessary information of the the given hostname's route
#result=''
#projects=$(oc get project -o jsonpath='{.items[*].metadata.name}')
#for project in $project; do
#  result=($(oc get -n $project routes --output="jsonpath={range .items[?(@.spec.host==\"$hostname\")]}{.spec.to.name};{.metadata.namespace};{.metadata.name};{.spec}{end}"))
#  if [ -n "${result}" ]; then
#    break
#  fi
#done
read service path termination < <(oc get -n $project route ${route} --output="jsonpath={.spec.to.name};{.spec.path};{.spec.tls..termination}")

IFS="$OIFS"

echo "Configuring certificate for requests to https://${hostname}${path}"

# Prepare key, cert and ca file to be inserted into json
key=$(sed ':a;N;$!ba;s/\n/\\n/g' $key_file)
cert=$(sed ':a;N;$!ba;s/\n/\\n/g' $cert_file)

issuer=$(openssl x509 -issuer -noout -in $cert_file)
ca_file="/usr/local/letsencrypt/ca/lets-encrypt-x${issuer#issuer= /C=US/O=Let\'s Encrypt/CN=Let\'s Encrypt Authority X}-cross-signed.pem"

if [[ -e $ca_file ]]; then
  ca=$(sed ':a;N;$!ba;s/\n/\\n/g' $ca_file)
else
  echo "ERROR: Could not determine issuing intermediate CA file. Tried \"$ca_file\"." >&2
fi

# Create backup of route's json definition, just in case
if [ ! -e "${project}_${route}.routebackup.json" ]; then
  oc export --namespace=$project routes $route --output=json > ${project}_${route}.routebackup.json
else
  oc export --namespace=$project routes $route --output=json > ${project}_${route}.routebackup.json.1
fi

# Modify the existing route
case $termination in
  edge|reencrypt)
    oc export --namespace=$project routes $route --output=json | jq " \
    .spec.tls.key=\"${key}\" | \
    .spec.tls.certificate=\"${cert}\" | \
    .spec.tls.caCertificate=\"${ca}\"" > \
    ${TMPDIR}/$route.new.json
    ;;
  passthrough)
    destination_ca=$(openssl s_client -connect ${hostname}:443 -servername ${hostname} -prexit -showcerts </dev/null 2>/dev/null | sed -nr '/BEGIN\ CERTIFICATE/H;//,/END\ CERTIFICATE/G;s/\n(\n[^\n]*){2}$//p' | sed ':a;N;$!ba;s/\n/\\n/g')

    if [ -n "$destination_ca" ]; then
      oc export --namespace=$project routes $route --output=json | jq " \
      .spec.tls.termination=\"reencrypt\" | \
      .spec.tls.key=\"${key}\" | \
      .spec.tls.certificate=\"${cert}\" | \
      .spec.tls.caCertificate=\"${ca}\" | \
      .spec.tls.destinationCACertificate=\"${destination_ca}\"" > \
      ${TMPDIR}/$route.new.json
    else
      echo "ERROR: Failed to obtain CA from backend. Route not replaced." >&2
      exit 1
    fi
    ;;
  *)
    oc export --namespace=$project routes $route --output=json | jq " \
    .spec.tls.termination=\"edge\" | \
    .spec.tls.key=\"${key}\" | \
    .spec.tls.certificate=\"${cert}\" | \
    .spec.tls.caCertificate=\"${ca}\" | \
    .spec.tls.insecureEdgeTerminationPolicy=\"Redirect\"" > \
    ${TMPDIR}/$route.new.json
    ;;
esac

if $dryrun; then
  echo -e "Dry-run enabled, old route not replaced. New route would look like this:\n"
  cat ${TMPDIR}/$route.new.json
else
  oc replace --namespace=$project routes $route -f ${TMPDIR}/$route.new.json
fi

