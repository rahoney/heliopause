load("//tools:defs.bzl", "cc_binary", "cc_library")

package(default_applicable_licenses = ["//:license"])

cc_binary(
    name = "haa_gvisor_observer",
    srcs = ["observer.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"7c6199801fd233d6d55309af4645d4746a077de7\\\""],
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/sentry/seccheck/points:points_cc_proto",
        "@com_google_protobuf//:protobuf",
    ],
)

cc_library(
    name = "observer_test_source",
    textual_hdrs = ["observer.cc"],
    visibility = ["//visibility:public"],
)

cc_binary(
    name = "haa_gvisor_observer_latch_test",
    srcs = ["observer_latch_test.cc"],
    defines = ["HAA_GVISOR_COMMIT=\\\"7c6199801fd233d6d55309af4645d4746a077de7\\\""],
    visibility = ["//visibility:public"],
    deps = [
		":observer_test_source",
        "//pkg/sentry/seccheck/points:points_cc_proto",
        "@com_google_protobuf//:protobuf",
    ],
)
