plugins {
    `java-library`
}


dependencies {
    implementation(project(":java-api"))

    implementation("io.opentracing:opentracing-api:0.33.0")
    compileOnly("io.opentracing.contrib:opentracing-grpc:0.2.3")

    api("com.nimbusds:oauth2-oidc-sdk:11.30.1")
    api(libs.typesafe.config)

    implementation(platform(libs.slf4j.bom))
    implementation("org.slf4j:slf4j-api")
    implementation(platform(libs.log4j.bom))
    implementation("org.apache.logging.log4j:log4j-api")
    implementation("org.apache.logging.log4j:log4j-core")
    implementation("org.apache.logging.log4j:log4j-slf4j-impl")

    testImplementation(platform(libs.grpc.bom))
    testImplementation("io.grpc:grpc-inprocess")
    testImplementation(platform(libs.junit.bom))
    testImplementation("org.junit.jupiter:junit-jupiter")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
    testImplementation("org.assertj:assertj-core:3.27.6")
    testImplementation(platform(libs.mockito.bom))
    testImplementation("org.mockito:mockito-core")
    testImplementation("org.mockito:mockito-junit-jupiter")
    testImplementation("org.mockito:mockito-inline:5.2.0")
}

tasks.test {
    useJUnitPlatform()
}
