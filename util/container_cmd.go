package util

import "fmt"

type ModContainerCmd struct {
}

func (ModContainerCmd) WithHostCurl(hostuserdir string) string {
	return fmt.Sprintf(`
mkdir -p /usr/lib /usr/lib64 /usr/bin
host_usr_dir=%s

copy_libs() {
  lib_name=$1
  # Find all matching libraries under host_usr_dir
  find "$host_usr_dir" -name "${lib_name}*" | while read src_file; do
    # Extract filename
    filename=$(basename "$src_file")
    # Determine target directory based on source path
    if [[ "$src_file" == *"lib64"* ]]; then
      cp "$src_file" "/usr/lib64/$filename" || true
    else
      cp "$src_file" "/usr/lib/$filename" || true
    fi
  done
}

# Copy required libraries
# for lib in libcurl libssl libcrypto libssh libldap liblber libsasl2 libnghttp libngtcp librtmp libps libgssapi libbrotli libhogweed libnettle libidn2 libunistring libtasn1 libkrb5 libzstd liblz libk5crypto libcom_err libkeyutils libheimbase libhx509 libselinux libcrypt libidn libheimntlm libntlm libnsl2 libresolv libkeyutils libhcrypto libsqlite3 libtirpc libwind libheimntlm libheimipcc libasn1 libheimipcc libas libroken; do
for lib in libcurl libssl libcrypto libssh libldap liblber libsasl2 libnghttp libngtcp librtmp libps libbrotli libhogweed libnettle libidn2 libunistring libtasn1 libkrb5 libzstd liblz libk5crypto libcom_err libkeyutils libheimbase libhx509 libselinux libcrypt libidn libheimntlm libntlm libnsl2 libresolv libkeyutils libhcrypto libsqlite3 libtirpc libwind libheimntlm libheimipcc libasn1 libheimipcc libas libroken libgssapi; do
  copy_libs "$lib"
done

# Find and copy curl binary
curl_path=$(find "$host_usr_dir" -name "curl" -type f -executable | head -n 1)
if [ -n "$curl_path" ]; then
  cp "$curl_path" /usr/bin/curl
else
  echo "curl binary not found in $host_usr_dir"
  exit 1
fi

curl -s http://%s:8003/bin_telego/install.sh | bash`, hostuserdir, MainNodeIp)
}
