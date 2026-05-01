#!/bin/bash
# Generate YANG modules for huawei package

set -e

YANG_DIR="../../yang-modules/network-router/8.20.10/ne40e-x8x16"
OUT_DIR="../internal/generated/huawei"

cd "$(dirname "$0")"

echo "Generating all YANG modules into all.gen.go..."
go run github.com/openconfig/ygot/generator \
    -path="$YANG_DIR" \
    -output_file="$OUT_DIR/all.gen.go" \
    -package_name=huawei \
    -generate_fakeroot=true \
    -compress_paths=false \
    huawei-vlan huawei-ifm huawei-system huawei-pub-type huawei-extension

cd $OUT_DIR

echo "Applying enum name fixes..."
sed -i '' 's/HuaweiIfm_PortType_50|100GE/HuaweiIfm_PortType_50_OR_100GE/g' ./all.gen.go
sed -i '' 's/HuaweiIfm_PortType_FlexE_50|100G/HuaweiIfm_PortType_FlexE_50_OR_100G/g' ./all.gen.go

echo "Formatting generated code..."
gofmt -w ./all.gen.go

echo "Done! Total lines: $(wc -l < all.gen.go)"
