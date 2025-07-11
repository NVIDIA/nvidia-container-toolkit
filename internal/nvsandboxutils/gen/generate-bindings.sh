#!/bin/bash
# Copyright 2024 NVIDIA CORPORATION
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This file generates bindings for nvsandboxutils by calling c-for-go.

set -x -e

PWD=$(pwd)
GEN_DIR="$PWD/gen"
PKG_DIR="$PWD"
GEN_BINDINGS_DIR="$GEN_DIR/nvsandboxutils"
PKG_BINDINGS_DIR="$PKG_DIR"

SOURCES=$(find "$GEN_BINDINGS_DIR" -type f)

mkdir -p "$PKG_BINDINGS_DIR"

cp "$GEN_BINDINGS_DIR/nvsandboxutils.h" "$PKG_BINDINGS_DIR/nvsandboxutils.h"
spatch --in-place --very-quiet --sp-file "$GEN_BINDINGS_DIR/anonymous_structs.cocci" "$PKG_BINDINGS_DIR/nvsandboxutils.h" > /dev/null

echo "Generating the bindings..."
c-for-go -out "$PKG_DIR/.." "$GEN_BINDINGS_DIR/nvsandboxutils.yml"
cd "$PKG_BINDINGS_DIR"
go tool cgo -godefs types.go > types_gen.go
go fmt types_gen.go
cd - > /dev/null
rm -rf "$PKG_BINDINGS_DIR/cgo_helpers.go" "$PKG_BINDINGS_DIR/types.go" "$PKG_BINDINGS_DIR/_obj"
go run "$GEN_BINDINGS_DIR/generateapi.go" --sourceDir "$PKG_BINDINGS_DIR" --output "$PKG_BINDINGS_DIR/zz_generated.api.go"
# go fmt "$PKG_BINDINGS_DIR"

SED_SEARCH_STRING='// WARNING: This file has automatically been generated on'
SED_REPLACE_STRING='// WARNING: THIS FILE WAS AUTOMATICALLY GENERATED.'
grep -l -R "$SED_SEARCH_STRING" "$PKG_DIR" | grep -v "/gen/" | xargs sed -i -E "s#$SED_SEARCH_STRING.*\$#$SED_REPLACE_STRING#g"

SED_SEARCH_STRING='// (.*) nvsandboxutils/nvsandboxutils.h:[0-9]+'
SED_REPLACE_STRING='// \1 nvsandboxutils/nvsandboxutils.h'
grep -l -RE "$SED_SEARCH_STRING" "$PKG_DIR" | grep -v "/gen/" | xargs sed -i -E "s#$SED_SEARCH_STRING\$#$SED_REPLACE_STRING#g"

