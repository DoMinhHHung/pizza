package com.pizza.auth.security;

import java.nio.charset.StandardCharsets;
import java.security.KeyFactory;
import java.security.interfaces.RSAPrivateKey;
import java.security.interfaces.RSAPublicKey;
import java.security.spec.PKCS8EncodedKeySpec;
import java.time.Instant;
import java.util.Base64;
import java.util.Date;
import java.util.UUID;

import org.springframework.stereotype.Service;

import com.pizza.auth.config.JwtProperties;
import com.pizza.auth.user.User;

import io.jsonwebtoken.Claims;
import io.jsonwebtoken.Jwts;

@Service
public class JwtService {

    private final JwtProperties jwtProperties;
    private final RSAPrivateKey privateKey;
    private final RSAPublicKey publicKey;

    public JwtService(JwtProperties jwtProperties) {
        this.jwtProperties = jwtProperties;
        String b64 = jwtProperties.getPrivateKeyBase64();
        if (b64 == null || b64.isBlank()) {
            throw new IllegalStateException(
                    "JWT_PRIVATE_KEY_BASE64 is not configured.");
        }
        this.privateKey = loadPrivateKey(b64);
        this.publicKey = derivePublicKey(this.privateKey);
    }

    public String getKeyId() {
        return jwtProperties.getKeyId();
    }

    public RSAPublicKey getPublicKey() {
        return publicKey;
    }

    public String generateAccessToken(User user, String jti) {
        Instant now = Instant.now();
        Instant expiry = now.plus(jwtProperties.getAccessTokenExpiration());
        return Jwts.builder()
                .id(jti)
                .header().keyId(jwtProperties.getKeyId()).and()
                .subject(user.getId().toString())
                .claim("email", user.getEmail())
                .claim("status", user.getStatus().name())
                .claim("username", user.getUsername())
                .claim("type", "access")
                .issuedAt(Date.from(now))
                .expiration(Date.from(expiry))
                .signWith(privateKey, Jwts.SIG.RS256)
                .compact();
    }

    public String generateRefreshToken(User user, String jti) {
        Instant now = Instant.now();
        Instant expiry = now.plus(jwtProperties.getRefreshTokenExpiration());
        return Jwts.builder()
                .id(jti)
                .header().keyId(jwtProperties.getKeyId()).and()
                .subject(user.getId().toString())
                .claim("type", "refresh")
                .issuedAt(Date.from(now))
                .expiration(Date.from(expiry))
                .signWith(privateKey, Jwts.SIG.RS256)
                .compact();
    }

    public Claims parseClaims(String token) {
        return Jwts.parser()
                .verifyWith(publicKey)
                .build()
                .parseSignedClaims(token)
                .getPayload();
    }

    public String getJti(String token) {
        return parseClaims(token).getId();
    }

    public UUID getUserId(String token) {
        return UUID.fromString(parseClaims(token).getSubject());
    }

    public boolean isAccessToken(String token) {
        return "access".equals(parseClaims(token).get("type", String.class));
    }

    public boolean isRefreshToken(String token) {
        return "refresh".equals(parseClaims(token).get("type", String.class));
    }

    public Instant getExpiration(String token) {
        return parseClaims(token).getExpiration().toInstant();
    }

    private static RSAPrivateKey loadPrivateKey(String base64Pem) {
        try {
            String pem = new String(Base64.getDecoder().decode(base64Pem.trim()), StandardCharsets.UTF_8);
            String normalized = pem
                    .replace("-----BEGIN PRIVATE KEY-----", "")
                    .replace("-----END PRIVATE KEY-----", "")
                    .replace("-----BEGIN RSA PRIVATE KEY-----", "")
                    .replace("-----END RSA PRIVATE KEY-----", "")
                    .replaceAll("\\s", "");
            byte[] der = Base64.getDecoder().decode(normalized);
            PKCS8EncodedKeySpec spec = new PKCS8EncodedKeySpec(der);
            return (RSAPrivateKey) KeyFactory.getInstance("RSA").generatePrivate(spec);
        } catch (Exception e) {
            throw new IllegalStateException(
                    "Failed to load JWT private key. Use PKCS#8 PEM Base64 (openssl pkcs8 -topk8 -nocrypt ...).", e);
        }
    }

    private static RSAPublicKey derivePublicKey(RSAPrivateKey privateKey) {
        try {
            var spec = new java.security.spec.RSAPublicKeySpec(
                    privateKey.getModulus(),
                    privateKey.getPublicExponent());
            return (RSAPublicKey) KeyFactory.getInstance("RSA").generatePublic(spec);
        } catch (Exception e) {
            throw new IllegalStateException("Failed to derive RSA public key", e);
        }
    }
}
