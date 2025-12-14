# syntax=docker/dockerfile:1.7

FROM eclipse-temurin:25-jdk AS builder
WORKDIR /app
ENV ORG_GRADLE_JAVA_HOME=/opt/java/openjdk \
    GRADLE_OPTS="-Dorg.gradle.java.home=/opt/java/openjdk"

COPY gradlew ./
COPY gradle ./gradle
COPY settings.gradle.kts build.gradle.kts gradle.properties ./
RUN chmod +x gradlew

COPY . .
RUN ./gradlew bootJar -x test --no-daemon

FROM eclipse-temurin:25-jre
WORKDIR /app

ENV SPRING_PROFILES_ACTIVE=prod \
    JAVA_TOOL_OPTIONS="--enable-native-access=ALL-UNNAMED --add-opens=java.base/java.util=ALL-UNNAMED --add-opens=java.base/java.lang=ALL-UNNAMED"

COPY --from=builder /app/build/libs/20q-kakao-bot-*.jar /app/app.jar
EXPOSE 8085

ENTRYPOINT ["java","-jar","/app/app.jar"]
