load("@bazel_gazelle//:def.bzl", "DEFAULT_LANGUAGES", "gazelle", "gazelle_binary")

gazelle_binary(
    name = "gazelle_binary",
    languages = DEFAULT_LANGUAGES + ["@golink//gazelle/go_link:go_default_library"],
    visibility = ["//visibility:public"],
)

# gazelle:prefix github.com/gcp-services/lock
gazelle(
    name = "gazelle",
    gazelle = "//:gazelle_binary",
)
