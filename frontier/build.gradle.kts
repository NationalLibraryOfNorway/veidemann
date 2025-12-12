plugins {
    `java`
    id("com.google.cloud.tools.jib")
}

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(21)
    }
}

dependencies {
    implementation(project(":java-api"))
    implementation(project(":commons"))
    implementation(project(":rethinkdbadapter"))

    implementation(libs.typesafe.config)
    implementation("redis.clients:jedis:7.1.0")

    // Concurrency limits
    implementation(libs.concurrency.limits.grpc)
    implementation(libs.concurrency.limits.core)

    // Prometheus
    implementation("io.prometheus:simpleclient:0.16.0")
    implementation("io.prometheus:simpleclient_hotspot:0.16.0")
    implementation("io.prometheus:simpleclient_httpserver:0.16.0")

    // gRPC
    implementation(platform(libs.grpc.bom))
    implementation("io.grpc:grpc-services")
    implementation("io.grpc:grpc-netty-shaded")

    // RethinkDB
    implementation("com.rethinkdb:rethinkdb-driver:2.3.3")
    implementation("com.google.code.gson:gson:2.13.2")
    implementation("org.yaml:snakeyaml:2.5")
    
    // Tracing
    implementation("io.jaegertracing:jaeger-client:1.8.1")
    implementation("io.opentracing.contrib:opentracing-grpc:0.2.3")
    implementation("io.opentracing:opentracing-noop:0.33.0")
    
    // Logging
    implementation(platform(libs.slf4j.bom))
    implementation("org.slf4j:slf4j-api")
    implementation(platform(libs.log4j.bom))
    implementation("org.apache.logging.log4j:log4j-api")
    implementation("org.apache.logging.log4j:log4j-core")
    implementation("org.apache.logging.log4j:log4j-slf4j2-impl")


    testImplementation(platform(libs.junit.bom))
    testImplementation("org.junit.jupiter:junit-jupiter")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")

    // Assert/Mockito/etc..
    testImplementation("org.assertj:assertj-core:3.24.2")
    testImplementation(platform(libs.mockito.bom))
    testImplementation("org.mockito:mockito-core")
    testImplementation("org.awaitility:awaitility:4.3.0")
    testImplementation("io.opentracing:opentracing-mock:0.33.0")
    
    // Testcontainers
    testImplementation("org.testcontainers:testcontainers-junit-jupiter:2.0.2")
}

tasks.test {
    useJUnitPlatform {
        excludeTags("integration")
    }
}

tasks.register<Test>("integrationTest") {
    useJUnitPlatform {
        includeTags("integration")
    }
    testClassesDirs = sourceSets["test"].output.classesDirs
    classpath = sourceSets["test"].runtimeClasspath
}

jib {
    to {
        image = "${rootProject.name}/${project.name}"
    }

    container {
        ports = listOf("8080", "5005")
    }
}
