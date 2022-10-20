export HOME="~"
export DEV_DIR="${HOME}/dev"
export BUFBUILD_BUF_DIR="${DEV_DIR}/buf"
export BUFBUILD_HOMEBREW_BUF_DIR="${DEV_DIR}/homebrew-buf"
export BUFBUILD_VIM_BUF_DIR="${DEV_DIR}/vim-buf"
export BUFBUILD_DOCS_BUF_BUILD_DIR="${DEV_DIR}/docs.buf.build"
export BUFBUILD_BUF_EXAMPLE_DIR="${DEV_DIR}/buf-examples"
export BUFBUILD_BUF_SETUP_ACTION_DIR="${DEV_DIR}/buf-setup-action"
export RELEASE_MINISIGN_PRIVATE_KEY="<key_value_from_1password>"
export RELEASE_MINISIGN_PRIVATE_KEY_PASSWORD="<password_value_from_1password>"

cd "${BUFBUILD_BUF_DIR}"
git switch main
git pull origin main
make installbuf
export VERSION=$(buf --version)

