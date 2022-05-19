#!/usr/bin/env bash

set -e

./compare.sh alpha registry token get buf.build --token-id deadbeef-dead-beef-dead-deadbeefdead
printf "deadbeef-dead-beef-dead-deadbeefdead" | ./compare.sh alpha registry token delete buf.build --token-id deadbeef-dead-beef-dead-deadbeefdead
./compare.sh beta registry commit get buf.build/sayers/petapis:test
./compare.sh beta registry commit list buf.build/sayers/norepo
./compare.sh beta registry commit list buf.build/sayers/petapis     # repo exist, but no commits
./compare.sh beta registry organization create buf.build/sma    # already exists
./compare.sh beta registry organization get buf.build/noorg
printf "noplugin" | ./compare.sh beta registry plugin delete buf.build/sayers/plugins/noplugin
./compare.sh beta registry plugin deprecate buf.build/sayers/plugins/nonexist
./compare.sh beta registry plugin undeprecate buf.build/sayers/plugins/nonexist
./compare.sh beta registry repository create buf.build/sayers/petapis --visibility=private    # already exists
./compare.sh beta registry repository get buf.build/sayers/norepo
printf "norepo" | ./compare.sh beta registry repository delete buf.build/sayers/norepo
./compare.sh beta registry repository deprecate buf.build/sayers/norepo
./compare.sh beta registry repository undeprecate buf.build/sayers/norepo
./compare.sh beta registry repository update buf.build/sayers/norepo --visibility=public
./compare.sh beta registry tag create buf.build/sayers/paymentapis:c43b40fb33b94296a9796f7bf733de5c paymenttag    # already exists
./compare.sh beta registry tag create buf.build/sayers/paymentapis:c43b40fb33b94296a9796f7bf7333333 paymenttag2   # commit does not exist
./compare.sh beta registry tag create buf.build/sayers/norepo:c43b40fb33b94296a9796f7bf7333333 paymenttag2    # repo does not exist
./compare.sh beta registry tag list buf.build/sayers/norepo
printf "notemplate" | ./compare.sh beta registry template delete buf.build/sayers/templates/notemplate
./compare.sh beta registry template deprecate buf.build/sayers/templates/notemplate
./compare.sh beta registry template undeprecate buf.build/sayers/templates/notemplate
printf "buf.build/sayers/norepo" | ./compare.sh beta registry track delete buf.build/sayers/norepo
printf "buf.build/sayers/petapis" | ./compare.sh beta registry track delete buf.build/sayers/petapis  # tried to delete main track, error as expected
./compare.sh beta registry track list buf.build/sayers/norepo


./compare.sh  beta registry organization create buf.build/sma
./compare.sh  beta registry organization get buf.build/sma
printf "sma" | ./compare.sh  beta registry organization delete buf.build/sma
./compare.sh  beta registry plugin list buf.build
./compare.sh  beta registry plugin version list buf.build/sayers/plugins/twirp
./compare.sh  beta registry plugin deprecate buf.build/sayers/plugins/twirp
./compare.sh  beta registry plugin undeprecate buf.build/sayers/plugins/twirp
printf "twirp" | ./compare.sh  beta registry plugin delete buf.build/sayers/plugins/twirp
./compare.sh  beta registry plugin create buf.build/sayers/plugins/twirp --visibility=private
./compare.sh  beta registry commit get buf.build/sayers/petapis
./compare.sh  beta registry commit list buf.build/sayers/petapis
./compare.sh  beta registry repository create buf.build/sayers/bufcli --visibility=private
./compare.sh  beta registry repository list buf.build
./compare.sh  beta registry repository get buf.build/sayers/bufcli
./compare.sh  beta registry repository deprecate buf.build/sayers/bufcli
./compare.sh  beta registry repository undeprecate buf.build/sayers/bufcli
./compare.sh  beta registry repository delete buf.build/sayers/bufcli
./compare.sh  beta registry repository update buf.build/sayers/bufcli --visibility=public
./compare.sh  beta registry tag create buf.build/sayers/paymentapis:<commit SHA> paymenttag
./compare.sh  beta registry tag list buf.build/sayers/paymentapis
./compare.sh  beta registry track list buf.build/sayers/petapis
./compare.sh  beta registry track delete buf.build/sayers/petapis
./compare.sh  beta registry template create buf.build/sayers/templates/twirp-go --visibility public --config '{"version":"v1","plugins":[{"owner":"library","name":"go","opt":["paths=source_relative"]},{"owner":"sayers","name":"twirp","opt":["paths=source_relative"]}]}'
./compare.sh  beta registry template list buf.build
./compare.sh  beta registry template deprecate buf.build/sayers/templates/twirp-go
./compare.sh  beta registry template undeprecate buf.build/sayers/templates/twirp-go
./compare.sh  beta registry template list buf.build
./compare.sh  beta registry template version create buf.build/sayers/templates/twirp-go --name v1 --config '{"version":"v1","plugin_versions":[{"owner":"library","name":"go","version":"v1.27.1-1"},{"owner":"sayers","name":"twirp","version":"v8.1.0-1"}]}'
./compare.sh  beta registry template version list buf.build/sayers/templates/twirp-go
./compare.sh  beta registry template delete buf.build/sayers/templates/testtemplate
./compare.sh  alpha protoc petapis
./compare.sh  alpha registry token create buf.build --note token-CLI
./compare.sh  alpha registry token list buf.build
./compare.sh  alpha registry token get buf.build --token-id <id>
./compare.sh  alpha registry token delete buf.build --token-id <id>