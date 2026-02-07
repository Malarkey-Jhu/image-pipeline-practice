```md
# Image Processing Pipeline（System Design 練習專案）PRD + Architecture + Testing Plan

> 目標：用一個「小而真實」的專案練 System Design，而不是做 UI/產品功能。  
> 核心練習：Async pipeline、MQ、Retry、Idempotency、Failure recovery、Consistency、Observability。

---

## 1. 專案目標（System Goals）

設計一個系統，允許使用者上傳圖片，並自動執行一系列處理（pipeline），最後讓使用者能看到最終圖片。

### 必須具備
- **非同步處理**：上傳與處理解耦（upload fast, process async）
- **高可靠**：worker crash / 網路抖動不丟任務
- **可重試**：暫時性錯誤可 retry（backoff）
- **幂等性**：同一任務重跑不會寫壞、不會產生衝突
- **可水平擴展**：worker 可 scale out
- **可觀測**：能定位任務卡哪、為何失敗、延遲如何

---

## 2. Out of Scope（刻意不做）

避免把時間浪費在細枝末節：
- ❌ Fancy UI / 前端框架（用 curl 或極簡頁面即可）
- ❌ 社交功能（like/comment/follow/feed）
- ❌ 搜尋、推薦、tagging
- ❌ 完整 auth 系統（可先用簡單 token 或省略）

---

## 3. Functional Requirements（功能需求）

### FR-1：取得上傳 URL（Pre-signed URL）
- Client 請求 API 取得 upload URL
- Client 直接上傳到 Object Storage（避免 API server 當帶寬瓶頸）

### FR-2：完成上傳後觸發 Pipeline
- Client 呼叫 complete-upload
- Server 建立處理任務（processing_task）並送 MQ
- Worker 依序處理每個 step（resize/compress/webp…）

### FR-3：查詢狀態
- Client 可查 media 狀態：
  - `PROCESSING` / `READY` / `FAILED`

### FR-4：取得最終圖片
- `READY` 時回傳 final image URL（CDN URL 或 signed GET）

---

## 4. Non-Functional Requirements（非功能需求）

### Reliability
- worker crash -> 不丟任務（MQ redelivery / DB lease recover）

### Idempotency
- 同一任務被重投/重跑，結果必須一致

### Scalability
- worker 可水平擴展，MQ 支援多 consumer

### Observability
- logs: 必須含 `media_id/task_id/worker_id/step`
- metrics: success rate、retry rate、latency、queue depth

---

## 5. 推薦架構（High-Level Architecture）

```

Client
↓
API Server (Go)
↓
Object Storage (MinIO/S3)  ← 存圖片（二進制）
↓
RabbitMQ                   ← 非同步任務
↓
Workers (Go)               ← 圖片處理
↓
Postgres                   ← 存 metadata / task 狀態

````

> 建議用 Docker Compose 一次啟動 Postgres + RabbitMQ + MinIO + API + Worker

---

## 6. API 設計（Minimal）

### 6.1 取得上傳 URL
**POST /upload-url**
Request（示例）
```json
{
  "content_type": "image/jpeg",
  "file_name": "a.jpg"
}
````

Response

```json
{
  "media_id": "01J...ULID",
  "upload_url": "https://minio/...presigned...",
  "original_key": "media/{media_id}/original.jpg",
  "expires_in": 300
}
```

### 6.2 完成上傳（觸發 pipeline）

**POST /complete-upload**

```json
{
  "media_id": "01J...",
  "original_key": "media/{media_id}/original.jpg"
}
```

Response

```json
{ "status": "PROCESSING" }
```

### 6.3 查詢 media 狀態

**GET /media/{media_id}**
Response

```json
{
  "media_id": "01J...",
  "status": "READY",
  "final_url": "https://cdn.example.com/media/{media_id}/final.webp"
}
```

---

## 7. DB Schema（推薦）

### 7.1 media

| column       | type      | note                                  |
| ------------ | --------- | ------------------------------------- |
| id           | ULID/UUID | media_id                              |
| status       | enum      | INIT/UPLOADED/PROCESSING/READY/FAILED |
| original_key | text      | object storage key                    |
| final_key    | text      | 最終版本 key（關鍵）                          |
| created_at   | timestamp |                                       |
| updated_at   | timestamp |                                       |

### 7.2 processing_task（核心）

| column      | type      | note                                   |
| ----------- | --------- | -------------------------------------- |
| id          | ULID/UUID | task id                                |
| media_id    | ULID/UUID |                                        |
| step        | enum      | resize/compress/webp/...               |
| status      | enum      | PENDING/RUNNING/SUCCEEDED/FAILED/RETRY |
| retry_count | int       |                                        |
| lock_by     | text      | worker_instance_id                     |
| lock_until  | timestamp | lease expiry                           |
| input_key   | text      | 來源 object key                          |
| output_key  | text      | 產出 object key                          |
| last_error  | text      | optional                               |
| created_at  | timestamp |                                        |
| updated_at  | timestamp |                                        |

#### 重要索引 / 約束

* `UNIQUE(media_id, step)`：保證同一步驟只存在一筆（幂等防線之一）
* `INDEX(status, lock_until)`：worker 扫描/claim 效率

---

## 8. Pipeline（建議先做 Linear）

例：

```
resize → compress → webp(final)
```

策略：

* 完成一步後，enqueue 下一步
* 所有必需 steps `SUCCEEDED` 後更新 `media.final_key` + `media.status=READY`

---

## 9. Worker 消費與狀態更新順序（非常重要）

### 9.1 Claim 任務（DB lease lock）

以「原子更新」方式 claim：

* 只有 `status in (PENDING, RETRY)` 且 `lock_until is null or expired` 才能被拿到
* 成功 claim 才做處理

### 9.2 正確順序（避免 ACK 後 GG）

1. claim task（DB）
2. download input (object storage)
3. process image
4. upload output (object storage)
5. update DB -> `SUCCEEDED`（或 `RETRY/FAILED`）
6. **ACK MQ**

> 原則：**先落盤（DB/狀態）再 ACK**
> 否則 ACK 完 crash，DB 沒寫到，就真的丟任務。

---

## 10. Retry 策略（建議）

* 暫時性錯誤：`RETRY` + exponential backoff + jitter
* 超過上限：`FAILED` + DLQ（可選）

建議 backoff：

* 1m, 5m, 30m, 2h（可依你喜好）

---

## 11. Idempotency（本專案的模板做法）

### 11.1 Output key deterministic（推薦）

每一步 output 固定 key，例如：

* `media/{media_id}/resize.jpg`
* `media/{media_id}/compress.jpg`
* `media/{media_id}/final.webp`

### 11.2 Worker 開始處理前可做 quick check（可選）

* 如果 output_key 已存在（HEAD object）

  * 直接更新 DB 為 `SUCCEEDED`（或 skip）
  * 這能抵抗重投 / 重跑

---

## 12. Testing Plan（你要怎麼測試）

> 測試重點不是 UI，而是：**可靠性 + 幂等 + failure recovery**。

### 12.1 Local 開發環境（Docker Compose）

建議 services：

* postgres
* rabbitmq (帶 management UI)
* minio
* api
* worker

啟動：

* `docker compose up -d`

驗證：

* RabbitMQ management：看 queue depth、unacked
* MinIO console：看 objects
* Postgres：看 media / processing_task 狀態

---

### 12.2 Happy Path 測試（端到端）

**目標：上傳 -> pipeline -> READY -> final_url 可下載**

步驟：

1. 呼叫 `POST /upload-url`
2. 用回傳的 `upload_url` PUT 上傳檔案（curl）
3. 呼叫 `POST /complete-upload`
4. 反覆 `GET /media/{id}` 直到 READY
5. 用 `final_url` 下載並驗證檔案存在

驗收標準：

* DB：所有 tasks SUCCEEDED
* MinIO：看到 original + outputs
* API：回 READY + final_url

---

### 12.3 幂等性測試（必做）

**Case A：重複呼叫 complete-upload**

* 對同一個 media_id 重複呼叫 `complete-upload` 2~3 次
* 期待：

  * processing_task 不會產生重複 row（UNIQUE(media_id, step)）
  * 不會產生多份 output objects（output_key deterministic）

**Case B：同一消息被重投**

* 手動讓 MQ 同一個 message 重複投遞（或不 ACK 讓它 requeue）
* 期待：

  * 最終只會得到一份 output
  * DB 狀態不會亂

---

### 12.4 Worker Crash 測試（最加分）

**目標：worker 在處理一半 crash，任務能被其他 worker 接手並成功完成**

方法：

1. 啟動 1 個 worker
2. 送一筆任務（complete-upload）
3. 等 worker 拿到任務後，立刻：

   * `docker kill worker_container`
4. 重啟 worker 或啟動第二個 worker
5. 期待：

   * 消息重新進 queue / 或 unacked 轉回 ready
   * 新 worker 重新 claim 任務
   * 最終 pipeline 完成

驗收標準：

* RabbitMQ：unacked 變回 ready（或重新投遞）
* DB：task 最終 SUCCEEDED（可能 retry_count +1）

---

### 12.5 ACK/DB 順序測試（非常有含金量）

**目標：驗證「先 DB 成功、再 ACK」的必要性**

做法（故意製造錯誤）：

* 在 worker 內部製造：

  * ACK 後立刻 panic/crash（或 kill -9）
  * 並讓 DB update 延後執行（模擬順序錯誤）

預期：

* 如果先 ACK：消息消失，但 DB 沒更新 -> 任務永遠卡住（BAD）
* 如果先 DB：即使 crash，消息重投也能被幂等處理（GOOD）

---

### 12.6 Timeout / Long Processing 測試（選做）

RabbitMQ 沒有像 SQS visibility timeout 那種 lease 延長 API；
RabbitMQ 的重投主要發生在：

* consumer 連線斷掉（消息回 queue）
* 或 consumer reject/nack

測試：

* 做一個 step 故意 sleep 很久（例如 2 min）
* 同時在 DB 做 lock_until 的 lease（例如 30s）
* 模擬「worker 很慢」時：

  * worker 是否會定期續 lease（更新 lock_until）
  * 避免被別的 worker 重複 claim（如果你有 DB polling/掃描）

---

## 13. 交付物（你練習時的“完成定義”）

* [ ] 端到端 happy path 跑通
* [ ] processing_task 狀態機完整（PENDING/RUNNING/SUCCEEDED/RETRY/FAILED）
* [ ] 幂等性測試：重跑不產生重複產物
* [ ] worker crash 測試：能恢復
* [ ] 至少有基本 metrics/logs（可選但非常加分）

---

## 14. 建議技術棧

* Go（API + Worker）
* RabbitMQ
* Postgres
* MinIO（S3 compatible）
* Docker Compose
* 圖片處理可先用簡單庫（不追求效果，追求流程）

---

## 15. V2 進階（完成第一版後再做）

* DLQ + 手動重放
* priority queue（不同 step/不同 media 優先級）
* outbox pattern（避免 DB & MQ 不一致）
* 多 variant（多尺寸、多格式）
* rate limiting（保護 worker）

