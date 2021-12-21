#!/bin/bash

echo 'Decoding with embedded DescriptorInfo...'
buf decode "$(cat pet.bin)" | jq

echo 'Decoding with module reference...'
buf decode --source buf.build/acme/petapis:84a33a06f0954823a6f2a089fb1bb82e --type pet.v1.Pet "$(cat pet.plain.bin)" | jq

echo 'Decoding with image reference...'
buf decode --source ../proto --type pet.v1.Pet "$(cat pet.plain.bin)" | jq

echo 'Encoding with module reference...'
buf encode '{"pet_id": "123", "name": "Ekans"}' --source buf.build/acme/petapis:84a33a06f0954823a6f2a089fb1bb82e --type pet.v1.Pet

echo 'Encoding with image reference...'
buf encode '{"pet_id": "123", "name": "Ekans"}' --source ../proto --type pet.v1.Pet

echo 'Round trip...'
buf encode '{"pet_id": "123", "name": "Ekans"}' --source buf.build/acme/petapis:84a33a06f0954823a6f2a089fb1bb82e --type pet.v1.Pet | buf decode | jq
