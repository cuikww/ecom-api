# Ecom API — Postman Collection (Modular)

Kenapa dipecah jadi banyak file? **Postman sendiri tidak punya fitur "collection lintas-file"** — satu collection yang bisa di-import selalu berupa **satu** file JSON. Yang bisa dibuat modular adalah *source*-nya: kamu edit request per fitur di file kecil-kecil, lalu satu script menggabungkannya jadi file final yang diimport ke Postman. Itu yang paket ini lakukan.

## Struktur folder

```
postman-modular/
├── collection.manifest.json      ← FILE INDUK. Cuma daftar folder role + file fitur mana yang digabung. Tidak ada request di sini.
├── build.js                      ← Script penggabung (Node.js, tanpa dependency)
├── package.json                  ← npm run build / npm run watch
├── collection/
│   ├── public/
│   │   ├── health.json           ← folder "Health"
│   │   ├── auth.json             ← folder "Auth" (register, login customer, login admin)
│   │   └── products.json         ← folder "Products (Read)"
│   ├── customer/
│   │   ├── profile.json          ← folder "Profile"
│   │   └── orders.json           ← folder "Orders"
│   └── admin/
│       ├── products.json         ← folder "Products (Manage)"
│       └── orders.json           ← folder "Orders (Manage)"
├── environment/
│   └── Ecom-API.postman_environment.json
└── dist/
    └── Ecom-API.postman_collection.json   ← HASIL GENERATE, ini yang di-import ke Postman
```

Struktur di Postman nanti:

```
Ecom API
├── Public
│   ├── Health
│   ├── Auth
│   └── Products (Read)
├── Customer
│   ├── Profile
│   └── Orders
└── Admin
    ├── Products (Manage)
    └── Orders (Manage)
```

## Cara pakai

### 1. Generate collection

```bash
node build.js
# atau
npm run build
```

Ini membaca `collection.manifest.json`, lalu tiap `features` di dalamnya dibaca dan digabung, hasilnya ditulis ke `dist/Ecom-API.postman_collection.json`.

### 2. Import ke Postman

- Import `dist/Ecom-API.postman_collection.json` → Collection
- Import `environment/Ecom-API.postman_environment.json` → Environment
- Pilih environment "Ecom API - Local" di kanan atas Postman

### 3. Jalankan alurnya

1. `Public > Auth > Register User` (opsional)
2. `Public > Auth > Login (Customer)` → `accessToken` & `customerId` otomatis tersimpan
3. Semua request di `Customer/*` otomatis pakai token itu
4. Untuk test folder `Admin/*`, jalankan `Public > Auth > Login (Admin)` dulu (butuh akun admin yang sudah di-seed di DB, karena endpoint register tidak menerima field `role`)
5. `productId`, `orderId` juga ke-set otomatis dari response berantai

### 4. Nambah/ubah fitur

**Ubah request yang sudah ada** → edit langsung file JSON-nya di `collection/<role>/<fitur>.json`, lalu `node build.js` lagi.

**Nambah fitur baru dalam role yang sama** (misal folder "Reviews" di bawah Customer):
1. Buat `collection/customer/reviews.json` dengan bentuk:
   ```json
   {
     "name": "Reviews",
     "description": "...",
     "item": [ /* request-request di sini */ ]
   }
   ```
2. Tambahkan pathnya ke `collection.manifest.json`, di `roles[].features` untuk role "Customer":
   ```json
   "features": [
     "collection/customer/profile.json",
     "collection/customer/orders.json",
     "collection/customer/reviews.json"
   ]
   ```
3. `node build.js`

**Nambah role baru** (misal "Superadmin") → tambah entry baru di `manifest.roles`, buat folder `collection/superadmin/`, isi feature file-nya, lalu build.

### 5. Auto-rebuild saat development (opsional)

```bash
npm run watch
```
Akan otomatis rebuild tiap ada perubahan file di `collection/`.

## Changelog terakhir (Products & Orders di-refactor jadi Category/Variant)

Kode `products` dan `orders` berubah struktur (harga & stok pindah ke level **variant**, produk sekarang punya Category/Images/Variants). Yang di-update di collection ini:

- **Public > Products (Read) > List Products**: sekarang juga meng-ekstrak `variantId` (dari `variants[0].id`) dan `categoryId`, bukan cuma `productId` — karena order butuh `variantId`.
- **Admin > Products (Manage)**: body Create/Update Product ditulis ulang total mengikuti `ProductParams` baru (`name`, `description`, `category_id`, `status`, `images[]`, `variants[]`). Ditambah request baru **Delete Product** (endpoint `DeleteProduct` baru muncul di service) + test idempotency.
- **Customer > Orders**: body `Place Order` sekarang pakai `variantId` (bukan `productId`), dan **wajib** menyertakan `customerId` di body (di-overwrite server, tapi lolos validasi `binding:"required"` butuh nilai != 0) — dipakai `{{customerId}}` otomatis.
- Environment: tambah variable `variantId` dan `categoryId` (default `1`, **asumsi** kategori dengan id 1 sudah ada di DB — sesuaikan kalau beda).

### ⚠️ Kemungkinan regresi yang saya temukan (bukan saya perbaiki, cuma ditandai di test)
Karena `handler.go`/`handlers.go` **tidak** ikut di-upload ulang saat kode service berubah, ada 2 kemungkinan mismatch antara handler lama dan service baru — saya tandai di collection dengan test yang menerima 2 kemungkinan status code + komentar `[Known issue]` / `[Cek regresi]`, supaya test tidak false-fail sampai kamu konfirmasi:

1. **UpdateProduct produk tidak ditemukan**: service sekarang tidak lagi convert `gorm.ErrRecordNotFound` → `ErrProductNotFound` di path Update (beda dari `FindProductByID` yang masih convert). Kalau handler masih cek `errors.Is(err, ErrProductNotFound)`, response yang tadinya 404 kemungkinan sekarang jadi 500.
2. **PlaceOrder variant tidak ditemukan**: service sekarang return `ErrorVariantNotFound` (bukan `ErrorProductNotFound` lagi). Kalau handler order masih cek `errors.Is(err, ErrorProductNotFound)`, response yang tadinya 404 kemungkinan sekarang jadi 500.

Begitu kamu re-upload `handler.go`/`handlers.go` versi terbaru, kabari saya — saya sesuaikan test-nya jadi strict lagi (cuma satu status code, tidak dua kemungkinan).

### ⚠️ Gotcha: Update Product bisa menghapus images/variants tanpa sengaja
Di `UpdateProduct`, `Association('Images').Clear()` dan `Association('Variants').Clear()` **selalu** jalan duluan sebelum diisi ulang dari body request. Kalau kamu PATCH tanpa field `images`/`variants`, semua images & variants produk itu akan **hilang** (bukan dibiarkan). Karena itu request `Update Product` di collection ini sengaja selalu mengirim `images` & `variants` lengkap — jangan dihapus manual kecuali kamu memang mau mengosongkannya.

## Catatan penting / asumsi

Karena file router (yang mendaftarkan path final + middleware auth per-role) tidak ikut diupload, beberapa hal di collection ini **asumsi** berdasarkan kode handler yang ada — silakan sesuaikan bagian `url` di file terkait kalau path aslinya beda:

- Endpoint auth: `POST /users`, `POST /users/login`
- Endpoint profile: `GET /users/me`, `PATCH /users/me` (pakai token, bukan `:id` di URL)
- Endpoint produk publik: `GET /products`, `GET /products/:id`
- Endpoint produk admin: `POST /products`, `PATCH /products/:id`
- Endpoint order customer: `POST /orders`, `GET /orders/customer/:customer_id`
- Endpoint order admin: `GET /orders`, `PATCH /orders/:id`
- Register (`POST /users`) hanya bisa membuat user dengan role `customer` (lihat `CreateUserParam`, tidak ada field `role`). Untuk test folder Admin, akun admin harus sudah ada di DB (mis. lewat seed/migration), lalu login lewat `Public > Auth > Login (Admin)`.
- Ditemukan kemungkinan bug kecil di `orders.Handler.UpdateOrderStatus`: cabang `ErrorOrderNotFound` saat ini mengembalikan HTTP **500**, bukan 404, meski pesan errornya "Pesanan tidak ditemukan". Test terkait (`Admin > Orders (Manage) > Update Order Status - Not Found`) sengaja menerima `[404, 500]` supaya tidak false-fail — sudah diberi komentar di request-nya, tinggal disesuaikan kalau sudah diperbaiki.
- `UpdateOrderStatusRequest.Status` valid values: `PENDING`, `SHIPPED`, `COMPLETED`, `CANCELED` (perhatikan ejaan **CANCELED**, bukan CANCELLED).
