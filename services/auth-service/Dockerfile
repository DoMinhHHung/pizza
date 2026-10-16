FROM eclipse-temurin:17-jdk-alpine AS build
WORKDIR /app
COPY mvnw .
COPY .mvn .mvn
COPY pom.xml .
RUN chmod +x mvnw \
    && ./mvnw -B -q dependency:go-offline -DskipTests
COPY src src
RUN ./mvnw -B -q package -DskipTests \
    && cp target/*.jar /app/app.jar

FROM eclipse-temurin:17-jre-alpine
WORKDIR /app
RUN apk add --no-cache curl \
    && addgroup -S pizza \
    && adduser -S pizza -G pizza
USER pizza
COPY --from=build /app/app.jar app.jar
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=5s --start-period=45s --retries=5 \
  CMD curl -fsS http://127.0.0.1:8080/actuator/health || exit 1
ENTRYPOINT ["java", "-XX:+UseContainerSupport", "-XX:MaxRAMPercentage=75.0", "-jar", "app.jar"]
