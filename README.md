# Parking Violation Portal

A microservices-based system for managing parking violations, billing, and payments.

## Architecture Overview

```
parking-violation-portal/
├── api-gateway/        # Reverse proxy & request router (Go)
├── backend/
│   ├── violation/      # Violation recording & lookup service (Go)
│   ├── billing/        # Invoice generation & management service (Go)
│   └── payment/        # Payment processing service (Go)
├── shared/             # Shared models, DB config, utilities (Go)
│   ├── models/
│   └── database/
├── frontend/           # Web portal UI (Next.js)
│   └── src/
├── docker-compose.yml  # Infrastructure: PostgreSQL, RabbitMQ, Redis
├── go.mod              # Go module root
└── README.md
```

## Tech Stack

| Layer        | Technology                     |
|--------------|--------------------------------|
| API Gateway  | Go (net/http / reverse proxy)  |
| Backend      | Go (microservices)             |
| Frontend     | Next.js 14 + TypeScript        |
| Database     | PostgreSQL 16                  |
| Message Queue| RabbitMQ 3.13                  |
| Cache        | Redis 7                        |
| Container    | Docker / Docker Compose        |

## Getting Started

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- [Go 1.22+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)

### Start Infrastructure

```bash
docker-compose up -d
```

| Service    | URL / Port                              |
|------------|-----------------------------------------|
| PostgreSQL | `localhost:5432`                        |
| RabbitMQ   | `localhost:5672` · UI: `localhost:15672`|
| Redis      | `localhost:6379`                        |

### Stop Infrastructure

```bash
docker-compose down
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/your-feature`)
3. Commit your changes (`git commit -m 'feat: add your feature'`)
4. Push to the branch (`git push origin feature/your-feature`)
5. Open a Pull Request

## Asumsi Sistem & Desain

Dalam pengerjaan proyek ini, beberapa asumsi dan keputusan desain penting diterapkan:
1. **Zona Waktu (Timezone)**: Zona waktu `Asia/Jakarta` secara ketat diatur dan diterapkan di Billing Service untuk mem-parsing timestamp pelanggaran secara akurat guna mengevaluasi aturan denda siang/malam.
2. **Kalkulasi Pelanggar Berulang (Repeat Offender)**: Sistem mendeteksi denda yang belum dibayar (*unpaid invoices*) dalam kurun waktu **90 hari terakhir** dari timestamp pelanggaran baru.
3. **Penguncian Versi Aturan (Version Locking)**: Nilai denda yang sudah terbit terikat secara permanen ke versi aturan (*FineRule Version ID*) saat pelanggaran tersebut dilaporkan. Publikasi aturan denda baru hanya memengaruhi pelanggaran baru, mencegah mutasi tidak sengaja pada data historis denda.
4. **Gateway Pembayaran Mock**: Skenario pembayaran dikendalikan secara parameter oleh klien melalui payload `scenario: "success"` atau `"failed"`, yang memicu respons simulasi sukses/gagal di Payment Service.

## Laporan Hasil Pengujian (Test Results Report)

Pengujian E2E (End-to-End) telah dilakukan pada lingkungan lokal dengan hasil sebagai berikut:

### 1. Pencatatan Pelanggaran & Penerbitan Invoice (Flow 1 & 2)
* **Langkah**: Mengisi form *Record New Violation* untuk plat nomor `B 1234 XYZ` dengan tipe `Expired Meter` pada waktu siang hari.
* **Hasil**: Pelanggaran sukses tercatat di DB, memicu penghitungan denda otomatis senilai Rp 50.000 (Base Rate v1), dan menghasilkan Invoice `INV-20260619-000002` dengan status `UNPAID` secara instan.
* **Status**: **LULUS (PASSED)**

### 2. Pengujian Denda Malam (Night-time Multiplier)
* **Langkah**: Mengisi form pelanggaran untuk plat nomor `B 7777 BBB` dengan memodifikasi timestamp ke pukul `23:30` (malam hari).
* **Hasil**: Billing Service menerapkan multiplier malam (1.5x) ke denda dasar, menghasilkan invoice dengan denda sebesar Rp 75.000 (Denda dasar Rp 50.000 x 1.5).
* **Status**: **LULUS (PASSED)**

### 3. Pengujian Pelanggar Berulang (Repeat Offender Multiplier)
* **Langkah**: Mengirimkan pelanggaran pertama untuk plat nomor `B 8888 AAA` dan membiarkannya tetap `UNPAID`. Kemudian, mengirimkan pelanggaran kedua untuk plat nomor `B 8888 AAA` yang sama.
* **Hasil**: Sistem mendeteksi adanya tagihan tidak lunas yang belum kedaluwarsa (<90 hari), lalu mengenakan Repeat Multiplier pada pelanggaran kedua tersebut secara otomatis.
* **Status**: **LULUS (PASSED)**

### 4. Pengujian Simulasi Pembayaran (Flow 4)
* **Langkah**: Melakukan lookup plat nomor `B 1234 XYZ` di Member Portal, mengeklik *Pay Now*, memilih skenario `success`, dan mengeklik *Settle Payment*.
* **Hasil**: Sistem berhasil memproses request melalui API Gateway ke Payment Service, menghasilkan `transaction_id` sukses dari gateway, dan memperbarui status invoice di database menjadi `PAID` secara real-time.
* **Status**: **LULUS (PASSED)**

### 5. Pengujian Kunci Versi Aturan (Flow 3 & 5)
* **Langkah**: Mengubah base rate denda `Expired Meter` dari Rp 50.000 menjadi Rp 100.000 di panel admin, mempublikasikannya (Ruleset v2), lalu membuat pelanggaran baru `B 5555 CCC` dengan tipe `Expired Meter`.
* **Hasil**: Pelanggaran baru terhitung Rp 100.000 (Ruleset v2). Di halaman utama (Transaction History), data historis `B 1234 XYZ` tetap terkunci pada denda Rp 50.000 (Version 1), sedangkan denda baru menggunakan Version 2.
* **Status**: **LULUS (PASSED)**

## License

MIT

