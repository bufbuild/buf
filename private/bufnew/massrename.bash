# arg1: find
# arg2: replace
# arg3: file
sed_file() {
  local tmpfile=./tmp.$$
  i="${3}"
  # could be a symlink directory
  if [ ! -d "${i}" ]; then
    if grep "${1}" "${i}" >/dev/null; then
      if [ -n "${DRY}" ]; then
        echo sed "s/${1}/${2}/g" "${i}"
      else
        sed "s/${1}/${2}/g" "${i}" > "${tmpfile}"
        mv "${tmpfile}" "${i}"
      fi
    fi
  fi
}

aggfiles() {
  ag \
    --nogroup \
    --ignore '*\.gen\.go' \
    --ignore '*\.pb\.go' \
    --ignore '*\.pb\.gw\.go' \
    --ignore '*\.pb\.validate\.go' \
    --ignore '*\.sql\.go' \
    --ignore '*\.twirp\.go' \
    --ignore 'gen/' \
    --ignore 'vendor/' \
    --file-search-regex '.go$' \
    --ignore private/bufpkg/bufmodule "${@}"| cut -f 1 -d : | sort | uniq
}

aggfiles moduleidentity | while IFS= read -r i; do
  sed_file 'bufmoduleref.ModuleIdentityForString' 'bufmodule.ParseModuleFullName' "${i}"
  sed_file 'bufmoduleref.NewModuleIdentity' 'bufmodule.NewModuleFullName' "${i}"
  sed_file 'bufmoduleref.ModuleIdentity' 'bufmodule.ModuleFullName' "${i}"
  sed_file 'private\/bufpkg\/bufmodule\/bufmoduleref' 'private\/bufnew\/bufmodule' "${i}"
  sed_file 'moduleIdentity.Remote()' 'moduleIdentity.Registry()' "${i}"
  sed_file 'moduleIdentity.Repository()' 'moduleIdentity.Name()' "${i}"
  sed_file 'moduleIdentity.IdentityString()' 'moduleIdentity.String()' "${i}"
  sed_file 'ModuleIdentity.IdentityString' 'ModuleFullName.String' "${i}"
  sed_file 'moduleIdentity' 'moduleFullName' "${i}"
  sed_file 'ModuleIdentity' 'ModuleFullName' "${i}"
  #sed_file 'WithModuleIdentityAndCommit' 'WithModuleFullNameAndCommitID' "${i}"
done

aggfiles moduleidentities | while IFS= read -r i; do
  sed_file 'moduleIdentities' 'moduleFullNames' "${i}"
  sed_file 'ModuleIdentities' 'ModuleFullNames' "${i}"
done

aggfiles --ignore private/buf/buffetch modulereference | while IFS= read -r i; do
  sed_file 'bufmoduleref.ModuleReferenceForString' 'bufmodule.ParseModuleRef' "${i}"
  sed_file 'bufmoduleref.NewModuleReference' 'bufmodule.NewModuleRef' "${i}"
  sed_file 'bufmoduleref.ModuleReference' 'bufmodule.ModuleRef' "${i}"
  sed_file 'private\/bufpkg\/bufmodule\/bufmoduleref' 'private\/bufnew\/bufmodule' "${i}"
  sed_file 'moduleReference.Remote()' 'moduleRef.Registry()' "${i}"
  sed_file 'moduleReference.Repository()' 'moduleRef.Name()' "${i}"
  sed_file 'SelectReferenceForRemote' 'SelectRefForRegistry' "${i}"
  sed_file 'NewModuleReference' 'NewModuleRef' "${i}"
  #sed_file 'moduleReference.ReferenceString()' 'moduleReference.String()' "${i}"
  sed_file 'moduleReference' 'moduleRef' "${i}"
  sed_file 'ModuleReference:' 'ModuleRef:' "${i}"
done
