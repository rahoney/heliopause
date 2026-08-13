load("//tools:defs.bzl", "cc_binary")

package(default_applicable_licenses = ["//:license"])

cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/sentry/seccheck/points:points_cc_proto",
        "@com_google_protobuf//:protobuf",
    ],
)
