# Executive Disorder API

This API provides access to summarized U.S. Executive Orders with socioeconomic impact analysis.

---

## 📡 Endpoints

### `GET /api/eos`
Returns a list of executive orders with optional filtering.

#### 🔍 Query Parameters

| Parameter  | Type   | Description                                                                 |
|------------|--------|-----------------------------------------------------------------------------|
| `president`| string | Case-insensitive substring match. E.g., `trump` matches "Donald Trump"      |
| `year`     | string | Filter by 4-digit year. E.g., `2024`                                         |
| `month`    | string | Filter by 2-digit month (with or without leading zero, e.g. `4` or `04`)     |
| `day`      | string | Filter by 2-digit day (with or without leading zero, e.g. `8` or `08`)       |

> ⛔ If `month` or `day` are provided without a corresponding `year`, the request is ignored for date filtering.

#### ✅ Example Request
```http
GET /api/eos?president=obama&year=2014&month=6&day=30
```

#### ✅ Example Response
```json
[
  {
    "eo_id": "2014-07999",
    "title": "Improving Federal Review of Infrastructure Projects",
    "date_issued": "2014-06-30",
    "president": "Barack Obama",
    "html_url": "...",
    "pdf_url": "...",
    "summary": [
      "Streamlines federal review of infrastructure projects",
      "Improves interagency coordination"
    ],
    "impact": {
      "average": "...",
      "poorest": "...",
      "richest": "..."
    }
  }
]
```

#### ❌ Example Error (No matches)
```json
{
  "error": "No matching executive orders found."
}
```

---

### `GET /api/eos/{eo_id}`
Returns a single executive order by ID.

#### ✅ Example Request
```http
GET /api/eos/2025-06380
```

#### ✅ Example Response
```json
{
  "eo_id": "2025-06380",
  "title": "Reinvigorating America's Beautiful Clean Coal Industry",
  "date_issued": "2025-01-24",
  "president": "Donald Trump",
  "html_url": "...",
  "pdf_url": "...",
  "summary": [
    "Declares coal a national priority",
    "Expands export opportunities"
  ],
  "impact": {
    "average": "...",
    "poorest": "...",
    "richest": "..."
  }
}
```

---

## 🛠️ Setup Notes

- Firestore DB: `eo-summary-db`
- Collection: `summaries`
- Project ID: `executive-disorder`

Make sure you’ve enabled the Firestore API and your credentials are authorized to access it.

---

## 📈 Coming Soon
- Filtering by topic/tag
- Pagination
- Admin API to reprocess documents

---