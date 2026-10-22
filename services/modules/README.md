# Auth Module – Pizza

Auth Module cung cấp backend xác thực cho end-user của ứng dụng khách trên nền tảng Pizza.

Module này hoàn toàn tách biệt với Platform Auth:

- Dữ liệu riêng (schema per project)
- Key pair ký token riêng
- Process runtime riêng (Rust)
- Token end-user không được Platform Auth / control-plane chấp nhận và ngược lại

## Mục tiêu

Người dùng Pizza bật Auth Module, cấu hình field + phương thức đăng nhập, deploy, sau đó nhận:

- URL backend
- URL Swagger (chỉ environment `dev`)
- URL JWKS

## Công nghệ

| Thành phần       | Lựa chọn                         |
| ---------------- | -------------------------------- |
| Ngôn ngữ         | Rust (edition 2021+)             |
| Web framework    | Axum                             |
| Async            | Tokio                            |
| Database         | PostgreSQL (Supabase) + SQLx     |
| Password hashing | bcrypt (cost ≥ 12)               |
| JWT              | RS256 / ES256 only               |
| Migration        | Chạy bởi deploy job (role riêng) |

## Kiến trúc ngắn

- **Path-based routing**: `/p/{project_id}/auth/...`
- **Schema isolation**: mỗi project một schema. Runtime dùng `SET LOCAL search_path`.
- **Hai role DB**:
  - `migration`: chỉ job deploy, có DDL
  - `runtime`: chỉ DML, search_path mặc định rỗng
- **Key management**: mỗi project một key pair. Private key được envelope encryption. Runtime nhận KEK qua secret môi trường để ký.
- **Config**: runtime chỉ đọc `published_configs` (schema publish riêng). Không đọc draft.

## Lát cắt triển khai

### Lát cắt 0a (nền tảng)

- Project resolution từ path
- Schema + hai role + test cô lập tenant
- Deploy job (claim, snapshot, migration, publish config)
- Sinh key pair + envelope encryption + JWKS
- Field Default + 1 Custom field
- `POST /register`, `POST /login`, `GET /me`

### Lát cắt 0b

- Refresh token rotation + reuse detection (có cửa sổ ân hạn)
- Logout
- Rate-limit (Postgres → 429 + Retry-After)
- Swagger dev + secret (có UI xem/đổi secret)

### Các lát cắt sau

1. userName (unique, case-insensitive, banned list)
2. Email verification + chống enumeration đầy đủ + outbox
3. Forgot / Reset password
4. resend + change-password
5. phoneNumber + Twilio
6. Redis (counter + invalidate config)
7. Progressive delay
8. Ẩn / xóa Custom field (2 bước)
9. Key rotation
10. Environment `prod`

## Field

### Default (khóa)

- `id`
- `emailAddress` (nullable)
- `passwordHash` (không bao giờ trả về API)
- `createdAt`
- `updatedAt`

### Built-in (bật/tắt)

- `phoneNumber` (E.164)
- `userName` (lát cắt sau)
- `fullName`, `address`, `avatarUrl`…

### System (ẩn)

- `emailVerified`, `phoneVerified`
- Refresh token storage
- OTP storage (hash)

### Custom

- `displayName` (UI) tách khỏi `apiKey` (API)
- Physical column = `cf_<immutable_id>`
- Chỉ cho sửa an toàn: displayName, validation, tăng độ dài
- Xóa 2 bước (ẩn → xóa vật lý)

## API chính (lá t cắt 0)

| Method | Path                                       | Mô tả                   |
| ------ | ------------------------------------------ | ----------------------- |
| POST   | /p/{project_id}/auth/register              | Đăng ký                 |
| POST   | /p/{project_id}/auth/login                 | Đăng nhập               |
| GET    | /p/{project_id}/auth/me                    | Thông tin user hiện tại |
| POST   | /p/{project_id}/auth/refresh               | Làm mới token (0b)      |
| POST   | /p/{project_id}/auth/logout                | Đăng xuất (0b)          |
| GET    | /p/{project_id}/auth/.well-known/jwks.json | JWKS                    |

## Bảo mật quan trọng

- Không tin `project_id` do client gửi.
- Mọi transaction runtime phải `SET LOCAL search_path`.
- password > 72 bytes → reject.
- Login/Forgot trả message chung + dummy bcrypt.
- Access token lifetime 30 phút (không thu hồi được trong khoảng này).
- Refresh 14 ngày, xoay vòng + reuse detection.
- Rate-limit trả 429, không sleep.
- Credential nhà cung cấp lưu mã hóa, chỉ tham chiếu bằng ID.
- OTP: worker sinh mã, lưu hash, không lưu plaintext.

## Deploy

- Một deploy tại một thời điểm per project + environment.
- Job lưu người deploy + thời điểm.
- Snapshot config bất biến.
- Thành công trả: Backend URL + Swagger URL + JWKS URL.

## Swagger

- Chỉ bật trên environment `dev`.
- Truy cập bằng secret theo environment.
- Có cơ chế xem / đổi secret trong UI (Owner/Admin).

## Ngoài phạm vi hiện tại

- Đổi kiểu field khi đã có dữ liệu
- Reference field liên module
- SMTP host tự do
- HS256
- Redis / RabbitMQ ở lát cắt 0
- Multi-environment ngay từ đầu

## Quy ước code

- Không comment bên trong thân function.
- Chỉ comment ngay trước function mô tả chức năng.
- Mọi đoạn code trước khi merge phải được phân tích ưu/nhược điểm; nhược điểm phải được sửa ngay.

## Trạng thái

Baseline đã khóa.
Hai điểm còn cần điền trước khi code lát cắt 0a:

1. Xác nhận runtime = Rust (đã chốt theo yêu cầu module).
2. Đối chiếu role thực tế trong control-plane (Owner/Admin/Member…).

Sau khi điền xong, có thể bắt đầu implement lát cắt 0a.
