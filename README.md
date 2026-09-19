# Konekuto OMS (Farm-to-Table Order Management System) 🥦🐟

Hệ thống quản lý đơn hàng backend cho mô hình nông sản tự trồng ("Từ vườn đến bàn ăn"), xây dựng bằng **Golang** theo chuẩn **Clean Architecture** và nguyên lý **SOLID** (100% Dependency Inversion qua Interfaces).

## 🛠 Tech Stack
- **Ngôn ngữ:** Golang 1.27
- **Web Framework:** Echo v5
- **Database:** SQLite (WAL mode) + `sqlx`
- **Security:** 
  - JWT Authentication (Access Token 15 phút & Refresh Token 7 ngày lưu DB).
  - RBAC (Role-Based Access Control) phân quyền `admin` / `customer`.
  - Token Blacklist thu hồi token khi logout kèm Background Worker dọn rác định kỳ.
- **Validation:** `go-playground/validator/v10`

---

## 🌾 Đặc thù Nghiệp vụ Nông sản (Farm-to-Table Domain)

- **Đơn vị tính linh hoạt (`unit`):** Bán theo quy cách cố định (`mớ`, `kg`, `túi 500g`, `con`) để đảm bảo số lượng (`quantity`) luôn là số nguyên, tránh sai số dấu phẩy động.
- **Quản lý vụ mùa (`status`):** Phân biệt trạng thái `active` (đang thu hoạch) và `out_of_season` (hết mùa/chờ lứa mới).
- **Gom đơn theo đợt (`delivery_date`):** Hỗ trợ chọn ngày giao để chủ vườn tổng hợp sản lượng và thu hoạch tươi trong ngày.
- **Hình thức nhận hàng (`order_type`):**
  - `pickup`: Nhận trực tiếp tại vườn (không bắt buộc địa chỉ giao).
  - `delivery`: Giao tận nơi (bắt buộc `shipping_address`).
- **Chống bán lố (Zero Overselling):** Sử dụng câu lệnh Atomic Update trực tiếp trong Database Transaction:
  ```sql
  UPDATE products 
  SET stock_quantity = stock_quantity - :quantity 
  WHERE id = :product_id AND stock_quantity >= :quantity AND deleted_at IS NULL

---

## 📊 Sơ đồ Nghiệp vụ (PlantUML Business Workflows)

### 1. Luồng Đặt Hàng & Xử Lý Giao Dịch (Order Checkout Flow)
Sơ đồ minh họa luồng xử lý Transaction khép kín qua các tầng Clean Architecture, bảo vệ tính toàn vẹn dữ liệu và cơ chế **Atomic Update** ngăn chặn bán vượt sản lượng.

```plantuml
@startuml
autonumber
skinparam style strictuml
skinparam SequenceMessageAlignment center

actor "Khách hàng\n(Customer)" as Client
participant "Echo Router\n& Middlewares" as Middleware
participant "Order Handler\n(Delivery Layer)" as Handler
participant "Order UseCase\n(Domain Logic)" as UseCase
participant "Product Repo" as ProductRepo
database "SQLite DB\n(WAL Mode)" as DB

Client -> Middleware: POST /api/v1/orders (Kèm JWT & Body)
activate Middleware
Middleware -> Middleware: Validate JWT & Check Blacklist
alt Token không hợp lệ / Đã bị thu hồi
    Middleware --> Client: 401 Unauthorized
end
Middleware -> Handler: Chuyển tiếp Request đã xác thực
deactivate Middleware

activate Handler
Handler -> Handler: Validate DTO (OrderType, DeliveryDate, Items)
Handler -> UseCase: CreateOrder(ctx, userID, req)
activate UseCase

loop Duyệt qua từng nông sản trong giỏ
    UseCase -> ProductRepo: GetByID(product_id)
    activate ProductRepo
    ProductRepo -> DB: SELECT * FROM products WHERE id = ?
    DB --> ProductRepo: Trả về thông tin sản phẩm
    ProductRepo --> UseCase: Product Entity
    deactivate ProductRepo

    alt Sản phẩm 'out_of_season' hoặc không tồn tại
        UseCase --> Handler: Lỗi (Hết mùa vụ / Không tìm thấy)
        Handler --> Client: 400 Bad Request
    end
    alt Số lượng đặt > stock_quantity
        UseCase --> Handler: Lỗi (Không đủ sản lượng vườn)
        Handler --> Client: 400 Bad Request
    end
    UseCase -> UseCase: Tính SubTotal = Price * Quantity
end

UseCase -> UseCase: Bắt đầu Transaction (BeginTxx)
UseCase -> DB: INSERT INTO orders (...)
loop Lưu từng OrderItem và Trừ Tồn Kho
    UseCase -> DB: INSERT INTO order_items (...)
    UseCase -> DB: UPDATE products SET stock_quantity = stock_quantity - :qty\nWHERE id = :id AND stock_quantity >= :qty
    alt RowsAffected == 0 (Kho bị âm hoặc có race condition)
        UseCase -> DB: ROLLBACK Transaction
        UseCase --> Handler: Lỗi (Hết hàng tại thời điểm chốt)
        Handler --> Client: 400 Bad Request
    end
end
UseCase -> DB: COMMIT Transaction
UseCase --> Handler: Order Entity hoàn tất
deactivate UseCase

Handler --> Client: 201 Created (Kèm chi tiết đơn hàng)
deactivate Handler
@enduml
```

---

### 2. Chu Trình Gom Đơn & Thu Hoạch (Farm-to-Table Lifecycle)
Sơ đồ hoạt động thể hiện vòng đời gom đơn theo ngày giao (`delivery_date`), phân loại hình thức nhận hàng (`pickup` / `delivery`) và quy trình thu hoạch tươi sống.

```plantuml
@startuml
start

:Khách hàng chọn nông sản theo quy cách (mớ, kg, túi, con);
:Khách chọn Ngày nhận (delivery_date) và Hình thức (order_type);

if (Hình thức nhận hàng?) then (Giao tận nơi - delivery)
  :Nhập địa chỉ giao (shipping_address);
else (Lấy tại vườn - pickup)
  :Bỏ qua địa chỉ, lấy tọa độ vườn;
endif

:Khách bấm Đặt hàng;

partition "Hệ thống Backend" {
  if (Sản phẩm còn trong mùa và đủ sản lượng?) then (Có)
    :Tạo đơn hàng trạng thái 'pending';
    :Trừ sản lượng tồn kho (Atomic Lock);
  else (Không)
    :Báo lỗi và hủy đơn (Rollback);
    stop
  endif
}

partition "Vận hành Nông Trại (Admin OMS)" {
  :Chủ vườn lọc danh sách đơn theo 'delivery_date';
  :Hệ thống tổng hợp sản lượng cần thu hoạch (vd: 30 mớ rau, 10 con cá);
  :Chủ vườn đổi trạng thái đơn sang 'confirmed';
  
  :Tiến hành thu hoạch sáng sớm tại vườn;
  :Sơ chế và đóng gói theo từng mã đơn hàng;
  
  if (order_type == 'pickup') then (Lấy tại vườn)
    :Chủ vườn đổi trạng thái sang 'ready';
    :Khách đến vườn nhận hàng;
  else (Giao tận nơi)
    :Chủ vườn đổi trạng thái sang 'shipping';
    :Đơn vị vận chuyển giao tận tay khách;
  endif
  
  :Đơn hàng hoàn tất -> Trạng thái 'completed';
}

stop
@enduml
```

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