plugins {
    `java-library`
    `application`
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


    api("com.rethinkdb:rethinkdb-driver:2.3.3")
    implementation("com.google.code.gson:gson:2.10.1")
    implementation("org.yaml:snakeyaml:2.0")
    
    // OpenTracing
    implementation("io.opentracing:opentracing-api:0.33.0")
    implementation("io.opentracing:opentracing-util:0.33.0")

    // Log4j and SLF4J
    implementation(platform(libs.slf4j.bom))
    implementation("org.slf4j:slf4j-api")
    implementation(platform(libs.log4j.bom))
    implementation("org.apache.logging.log4j:log4j-api")
    implementation("org.apache.logging.log4j:log4j-core")
    implementation("org.apache.logging.log4j:log4j-slf4j2-impl")

    // Mockito
    testImplementation(platform(libs.mockito.bom))
    testImplementation("org.mockito:mockito-core")
    // AssertJ
    testImplementation("org.assertj:assertj-core:3.27.6")
    // JUnit
    testImplementation(platform(libs.junit.bom))
    testImplementation("org.junit.jupiter:junit-jupiter")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
}

application {
    mainClass.set("no.nb.nna.veidemann.db.initializer.Main")
}

jib {
    to {
        image = "${rootProject.name}/db-initializer"
    }
}

tasks.test {
    useJUnitPlatform()
}
