plugins {
    `java-library`
}


dependencies {
    implementation(project(":java-api"))

    implementation(libs.opentracing.api)
    compileOnly(libs.opentracing.grpc)

    api(libs.oauth2.oidc.sdk)
    api(libs.typesafe.config)

    implementation(platform(libs.slf4j.bom))
    implementation("org.slf4j:slf4j-api")
    implementation(platform(libs.log4j.bom))
    implementation("org.apache.logging.log4j:log4j-api")
    implementation("org.apache.logging.log4j:log4j-core")
    implementation("org.apache.logging.log4j:log4j-slf4j2-impl")

    testImplementation(platform(libs.grpc.bom))
    testImplementation("io.grpc:grpc-inprocess")
    testImplementation(platform(libs.junit.bom))
    testImplementation("org.junit.jupiter:junit-jupiter")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
    testImplementation(platform(libs.assertj.bom))
    testImplementation("org.assertj:assertj-core")
    testImplementation(platform(libs.mockito.bom))
    testImplementation("org.mockito:mockito-core")
    testImplementation("org.mockito:mockito-junit-jupiter")
}

tasks.test {
    useJUnitPlatform()
}
