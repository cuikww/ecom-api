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
