load("//tools:defs.bzl", "cc_binary")

package(default_applicable_licenses = ["//:license"])

cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"5ceb9a5fd5750d6c73dd166441f28306039300d0\\\""],
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/sentry/seccheck/points:points_cc_proto",
        "@com_google_protobuf//:protobuf",
    ],
)
