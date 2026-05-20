plugins {
    `java-library`
    `application`
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


    api(libs.rethinkdb.driver)
    implementation(libs.gson)
    implementation(libs.snakeyaml)
    
    // OpenTracing
    implementation(libs.opentracing.api)
    implementation(libs.opentracing.util)

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
    testImplementation(platform(libs.assertj.bom))
    testImplementation("org.assertj:assertj-core")
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
