import com.google.protobuf.gradle.*

plugins {
    id("java-library")
    alias(libs.plugins.protobuf)
}

dependencies {
    api(libs.protobuf.java.util)
    api(libs.grpc.protobuf)
    api(libs.grpc.stub)
}

sourceSets {
    val main by getting {
        proto {
            srcDir(rootDir.resolve("proto"))
        }
    }
}

protobuf {
    protoc {
        artifact = "com.google.protobuf:protoc:${libs.versions.protobuf.get()}"
    }

    plugins {
        id("grpc") {
            artifact = "io.grpc:protoc-gen-grpc-java:${libs.versions.grpc.get()}"
        }
    }

    generateProtoTasks {
        ofSourceSet("main").configureEach {
            plugins {
                id("grpc")
            }
        }
    }
}
