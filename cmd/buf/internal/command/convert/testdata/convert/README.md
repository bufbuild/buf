To re-generate the `descriptor.plain.binpb` file, run the following command in the root of the project:

```
echo "{\"one\":\"55\"}" | \
buf convert private/buf/cmd/buf/testdata/success \
    --type buf.Foo \
    --from -#format=json \
    --to private/buf/cmd/buf/command/convert/testdata/convert/descriptor.plain.binpb
```

To re-generate the `bin_json/duration.{bin,binpb,json,txtpb}` files, run the following command in the root of the project:

```
for EXT in bin binpb json txtpb
do
    echo "\"3600s\"" | \
    buf convert \
        --type google.protobuf.Duration \
        --from -#format=json \
        --to private/buf/cmd/buf/command/convert/testdata/convert/bin_json/duration.$EXT
done
```

To re-generate the `bin_json/image.{bin,binpb,json,txtpb}` files, run the following command in the root of the project:

```
for EXT in bin binpb json txtpb
do
    buf build private/buf/cmd/buf/command/convert/testdata/convert/bin_json/buf.proto \
        --output private/buf/cmd/buf/command/convert/testdata/convert/bin_json/image.$EXT
done
```

To re-generate the `bin_json/payload.{bin,binpb,json,txtpb}` files, run the following command in the root of the project:

```
for EXT in bin binpb json txtpb
do
    echo "{\"one\":\"55\"}" | \
    buf convert private/buf/cmd/buf/command/convert/testdata/convert/bin_json \
        --type buf.Foo \
        --from -#format=json \
        --to private/buf/cmd/buf/command/convert/testdata/convert/bin_json/payload.$EXT
done
```

