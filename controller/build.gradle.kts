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

    implementation(libs.jaeger.client)

    implementation(platform(libs.grpc.bom))
    implementation("io.grpc:grpc-netty-shaded")

    implementation(libs.checker.qual)
    implementation(libs.cron4j)
    
    implementation(libs.opentracing.grpc)
    implementation(libs.opentracing.noop)

    implementation(libs.typesafe.config)

    implementation(platform(libs.slf4j.bom))
    implementation("org.slf4j:slf4j-api")
    implementation("org.slf4j:jul-to-slf4j")
    implementation(platform(libs.log4j.bom))
    implementation("org.apache.logging.log4j:log4j-api")
    implementation("org.apache.logging.log4j:log4j-core")
    implementation("org.apache.logging.log4j:log4j-slf4j2-impl")

    testImplementation("io.grpc:grpc-inprocess")
    testImplementation(platform(libs.junit.bom))
    testImplementation(platform(libs.assertj.bom))
    testImplementation("org.junit.jupiter:junit-jupiter")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
    testImplementation("org.assertj:assertj-core")
    testImplementation(platform(libs.mockito.bom))
    testImplementation("org.mockito:mockito-core")
}

tasks.test {
    useJUnitPlatform()
}

jib {
    to {
        image = "${rootProject.name}/${project.name}"
    }

    container {
        ports = listOf("50051")
    }
}
