version: v1
lint:
  use:
    - PACKAGE_DIRECTORY_MATCH
    - ENUM_PASCAL_CASE
    - FIELD_LOWER_SNAKE_CASE
    - MESSAGE_PASCAL_CASE
  ignore_only:
    ENUM_PASCAL_CASE:
      - buf/bar/bar.proto
      - buf/foo/bar
      - buf/foo/bar
    MESSAGE_PASCAL_CASE:
      - buf/bar/bar.proto
    BASIC:
      - buf/foo/bar
