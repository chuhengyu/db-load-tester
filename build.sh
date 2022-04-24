#! /bin/bash
# Cross-platform build

v=v101
output_name=db-load-tester
os=linux
arch=amd64
if [[ $1 = "linux" ]]; then
  os=linux
elif [[ $1 = "macos" ]]; then
  os=darwin
else
  echo "[WARN] unrecognized or input for platform. Usage: ./build.sh (linux|macos)"
  echo "[INFO] Building with current platform's spec. Output: ${output_name}"
  go build -o ${output_name}
  exit 1 
fi

echo "building with ${1} spec. Output: ${output_name}-${1}-${v}"
GOOS=${os} GOARCH=${arch} go build -o ${output_name}-${1}-${v}
