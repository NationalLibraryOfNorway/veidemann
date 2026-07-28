plugins {
    `java-library`
    `application`
    alias(libs.plugins.jib)
}

val mainSourceSet = sourceSets.getByName("main")

val integrationTest = sourceSets.create("integrationTest") {
    compileClasspath +=
        mainSourceSet.output +
            configurations.testRuntimeClasspath.get()

    runtimeClasspath += output + compileClasspath
}

configurations[integrationTest.implementationConfigurationName].extendsFrom(configurations.testImplementation.get())
configurations[integrationTest.runtimeOnlyConfigurationName].extendsFrom(configurations.testRuntimeOnly.get())

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

val integrationTestSystemProperties = listOf(
    "db.host",
    "db.port",
    "db.name",
    "db.user",
    "db.password"
)

tasks.register<Test>("integrationTest") {
    description = "Runs rethinkdbadapter integration tests against a live RethinkDB instance"
    group = "verification"
    testClassesDirs = integrationTest.output.classesDirs
    classpath = integrationTest.runtimeClasspath
    shouldRunAfter(tasks.test)
    useJUnitPlatform()

    integrationTestSystemProperties.forEach { propertyName ->
        System.getProperty(propertyName)?.let { propertyValue ->
            systemProperty(propertyName, propertyValue)
        }
    }

    if (!systemProperties.containsKey("db.name")) {
        systemProperty("db.name", "test")
    }

    doFirst {
        val missingProperties = listOf("db.host", "db.port").filterNot(systemProperties::containsKey)
        check(missingProperties.isEmpty()) {
            "Missing required system properties for integrationTest: ${missingProperties.joinToString(", ")}. " +
                    "Run with -Ddb.host=<host> -Ddb.port=<port>."
        }
    }
}
