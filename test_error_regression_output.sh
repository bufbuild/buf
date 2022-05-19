#!/usr/bin/env bash

set -e

./compare.sh alpha registry token get buf.build --token-id deadbeef-dead-beef-dead-deadbeefdead
printf "deadbeef-dead-beef-dead-deadbeefdead" | ./compare.sh alpha registry token delete buf.build --token-id deadbeef-dead-beef-dead-deadbeefdead
./compare.sh beta registry commit get buf.build/achiu/petapis:test
./compare.sh beta registry commit list buf.build/achiu/norepo
./compare.sh beta registry commit list buf.build/achiu/petapis     # repo exist, but no commits
./compare.sh beta registry organization create buf.build/sma    # already exists
./compare.sh beta registry organization get buf.build/noorg
printf "noplugin" | ./compare.sh beta registry plugin delete buf.build/achiu/plugins/noplugin
./compare.sh beta registry plugin deprecate buf.build/achiu/plugins/nonexist
./compare.sh beta registry plugin undeprecate buf.build/achiu/plugins/nonexist
./compare.sh beta registry repository create buf.build/achiu/petapis --visibility=private    # already exists
./compare.sh beta registry repository get buf.build/achiu/norepo
printf "norepo" | ./compare.sh beta registry repository delete buf.build/achiu/norepo
./compare.sh beta registry repository deprecate buf.build/achiu/norepo
./compare.sh beta registry repository undeprecate buf.build/achiu/norepo
./compare.sh beta registry repository update buf.build/achiu/norepo --visibility=public
./compare.sh beta registry tag create buf.build/achiu/paymentapis:c43b40fb33b94296a9796f7bf733de5c paymenttag    # already exists
./compare.sh beta registry tag create buf.build/achiu/paymentapis:c43b40fb33b94296a9796f7bf7333333 paymenttag2   # commit does not exist
./compare.sh beta registry tag create buf.build/achiu/norepo:c43b40fb33b94296a9796f7bf7333333 paymenttag2    # repo does not exist
./compare.sh beta registry tag list buf.build/achiu/norepo
printf "notemplate" | ./compare.sh beta registry template delete buf.build/achiu/templates/notemplate
./compare.sh beta registry template deprecate buf.build/achiu/templates/notemplate
./compare.sh beta registry template undeprecate buf.build/achiu/templates/notemplate
printf "buf.build/achiu/norepo" | ./compare.sh beta registry track delete buf.build/achiu/norepo
printf "buf.build/achiu/petapis" | ./compare.sh beta registry track delete buf.build/achiu/petapis  # tried to delete main track, error as expected
./compare.sh beta registry track list buf.build/achiu/norepo


./compare.sh  beta registry organization create buf.build/sma
./compare.sh  beta registry organization get buf.build/sma
printf "sma" | ./compare.sh  beta registry organization delete buf.build/sma
./compare.sh  beta registry plugin list buf.build
./compare.sh  beta registry plugin version list buf.build/achiu/plugins/twirp
printf "twirp" | ./compare.sh  beta registry plugin delete buf.build/achiu/plugins/twirp
./compare.sh  beta registry plugin create buf.build/achiu/plugins/twirp --visibility=private
./compare.sh  beta registry commit get buf.build/achiu/petapis
./compare.sh  beta registry commit list buf.build/achiu/petapis
./compare.sh  beta registry repository create buf.build/achiu/bufcli --visibility=private
./compare.sh  beta registry repository list buf.build
./compare.sh  beta registry repository get buf.build/achiu/bufcli
printf "bufcli" | ./compare.sh  beta registry repository delete buf.build/achiu/bufcli
./compare.sh  beta registry repository update buf.build/achiu/bufcli --visibility=public
# ./compare.sh  beta registry tag create buf.build/achiu/paymentapis:<commit SHA> paymenttag # TODO: this isn't necessarily working for me just yet
./compare.sh  beta registry tag list buf.build/achiu/paymentapis
./compare.sh  beta registry track list buf.build/achiu/petapis
printf "buf.build/achiu/petapis" | ./compare.sh  beta registry track delete buf.build/achiu/petapis
./compare.sh  beta registry template create buf.build/achiu/templates/twirp-go --visibility public --config '{"version":"v1","plugins":[{"owner":"library","name":"go","opt":["paths=source_relative"]},{"owner":"achiu","name":"twirp","opt":["paths=source_relative"]}]}'
./compare.sh  beta registry template list buf.build
./compare.sh  beta registry template list buf.build
./compare.sh  beta registry template version create buf.build/achiu/templates/twirp-go --name v1 --config '{"version":"v1","plugin_versions":[{"owner":"library","name":"go","version":"v1.27.1-1"},{"owner":"achiu","name":"twirp","version":"v8.1.0-1"}]}'
./compare.sh  beta registry template version list buf.build/achiu/templates/twirp-go
printf "testtemplate" | ./compare.sh  beta registry template delete buf.build/achiu/templates/testtemplate
./compare.sh  alpha protoc petapis
./compare.sh  alpha registry token list buf.build
./compare.sh  alpha registry token get buf.build --token-id 41746a4b-47cd-4ae0-82d1-9fdebde113c4

# The following is going to be noisy. Maybe we can do this manually?
# ./compare.sh  alpha registry token create buf.build --note token-CLI # This is going to be different since the output returned is a generated sha
# ./compare.sh  alpha registry token delete buf.build --token-id <id>

#./compare.sh  beta registry repository deprecate buf.build/achiu/bufcli
#./compare.sh  beta registry repository undeprecate buf.build/achiu/bufcli

#./compare.sh  beta registry template deprecate buf.build/achiu/templates/twirp-go
#./compare.sh  beta registry template undeprecate buf.build/achiu/templates/twirp-go

#./compare.sh  beta registry plugin deprecate buf.build/achiu/plugins/twirp
#./compare.sh  beta registry plugin undeprecate buf.build/achiu/plugins/twirp
