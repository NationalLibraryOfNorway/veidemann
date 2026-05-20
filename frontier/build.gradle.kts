plugins {
    `java`
    alias(libs.plugins.jib)
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
    implementation(libs.jedis)

    // Prometheus
    implementation(libs.prometheus.simpleclient)
    implementation(libs.prometheus.simpleclient.hotspot)
    implementation(libs.prometheus.simpleclient.httpserver)

    // gRPC
    implementation(platform(libs.grpc.bom))
    implementation("io.grpc:grpc-services")
    implementation("io.grpc:grpc-netty-shaded")

    // RethinkDB
    implementation(libs.rethinkdb.driver)
    implementation(libs.gson)
    implementation(libs.snakeyaml)
    
    // Tracing
    implementation(libs.jaeger.client)
    implementation(libs.opentracing.grpc)
    implementation(libs.opentracing.noop)
    
    // Logging
    implementation(platform(libs.slf4j.bom))
    implementation("org.slf4j:slf4j-api")
    implementation("org.slf4j:jul-to-slf4j")
    implementation(platform(libs.log4j.bom))
    implementation("org.apache.logging.log4j:log4j-api")
    implementation("org.apache.logging.log4j:log4j-core")
    implementation("org.apache.logging.log4j:log4j-slf4j2-impl")


    testImplementation(platform(libs.junit.bom))
    testImplementation("org.junit.jupiter:junit-jupiter")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")

    // Assert/Mockito/etc..
    testImplementation(platform(libs.assertj.bom))
    testImplementation("org.assertj:assertj-core")
    testImplementation(platform(libs.mockito.bom))
    testImplementation("org.mockito:mockito-core")
    testImplementation(libs.awaitility)
    testImplementation(libs.opentracing.mock)
    
    // Testcontainers
    testImplementation(libs.testcontainers.junit.jupiter)
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
