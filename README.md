# Konekuto OMS (Order Management System) 🚀

Hệ thống quản lý đơn hàng (E-commerce) backend được xây dựng với các tiêu chuẩn khắt khe nhất dành cho môi trường Production, tuân thủ tuyệt đối **Clean Architecture** và nguyên lý **SOLID** (100% Dependency Inversion qua Interfaces).

## 🛠 Tech Stack (Công nghệ sử dụng)
- **Ngôn ngữ:** Golang 1.27
- **Web Framework:** Echo v5 (Hiệu năng cao, routing tối ưu)
- **Database:** SQLite (WAL mode) + `sqlx` (Giao tiếp DB an toàn, chống SQL Injection)
- **Security:** 
  - JWT (JSON Web Token) cho Authentication (Access & Refresh Token).
  - RBAC (Role-Based Access Control) cho Phân quyền (Admin/Customer).
  - Bcrypt băm mật khẩu.
  - Blacklist Token (Thu hồi token khi Logout).
- **Validation:** `go-playground/validator/v10`

---

## 🏗 Kiến trúc Hệ thống (Clean Architecture Flow)

Dự án được chia thành các Layer (Tầng) độc lập. Các tầng giao tiếp với nhau **hoàn toàn thông qua Interfaces**, giúp mã nguồn dễ dàng viết Unit Test và dễ dàng thay đổi công nghệ (VD: Đổi SQLite sang PostgreSQL mà không cần sửa Core Logic).

**Luồng đi của một Request (API Flow):**
1. **Client** gửi HTTP Request (JSON).
2. **Server / Middleware:** Bắt Request, kiểm tra CORS, Rate Limit, ghi Log, và xác thực JWT (nếu API bị khóa).
3. **Delivery (Handler):** Nhận Request, parse JSON, Validate dữ liệu đầu vào.
4. **UseCase (Domain Logic):** Nhận DTO từ Handler, xử lý nghiệp vụ lõi (tính toán, cấp UUID, kiểm tra điều kiện).
5. **Repository:** Được UseCase gọi để lưu/lấy dữ liệu. Trực tiếp thực thi câu lệnh SQL với Database.
6. Kết quả đi ngược từ dưới lên và trả về Client (chuẩn JSON Response).

---

## 📂 Cấu trúc Thư mục (Directory Structure)

```text
konekuto-oms/
├── cmd/
│   └── api/
│       └── main.go              # Điểm khởi chạy của ứng dụng (Entry point)
├── configs/
│   └── config.local.yaml        # File cấu hình (Port, DB, Secret - Được Ignore)
├── internal/
│   ├── config/                  # Load cấu hình bằng Viper
│   ├── domain/                  # Lõi hệ thống: Entities, DTOs, Interfaces
│   ├── server/                  # Quản lý vòng đời Server & Dependency Injection
│   ├── product/                 # Module Sản phẩm (Handler, UseCase, Repository)
│   └── user/                    # Module Người dùng (Handler, UseCase, Repository)
├── pkg/
│   ├── middlewares/             # Các Middleware tự viết (Auth, RBAC, Blacklist)
│   ├── response/                # Chuẩn hóa JSON Response trả về Client
│   └── validations/             # Cấu hình Validator
├── scripts/
│   └── init.sql                 # Script tạo DB Schema ban đầu
├── .gitignore
├── go.mod
└── README.md