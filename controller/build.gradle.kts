plugins {
    `java`
    id("com.google.cloud.tools.jib") version "3.5.1"
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

    implementation(libs.concurrency.limits.core)
    implementation(libs.concurrency.limits.grpc)
    implementation(libs.jaeger.client)

    implementation(platform(libs.grpc.bom))
    implementation("io.grpc:grpc-inprocess")

    implementation("org.checkerframework:checker-qual:3.52.0")
    implementation("it.sauronsoftware.cron4j:cron4j:2.2.5")
    
    implementation("io.opentracing.contrib:opentracing-grpc:0.2.3")
    implementation("io.opentracing:opentracing-noop:0.33.0")

    implementation(libs.typesafe.config)

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
    testImplementation("org.assertj:assertj-core:3.24.2")
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
