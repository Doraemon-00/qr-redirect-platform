import http from "k6/http";
import { check } from "k6";

export const options = {
  scenarios: {
    create_qr: {
      executor: "constant-arrival-rate",
      rate: Number(__ENV.RATE || 10),
      timeUnit: "1s",
      duration: __ENV.DURATION || "30s",
      preAllocatedVUs: Number(__ENV.VUS || 20),
      maxVUs: Number(__ENV.MAX_VUS || 100),
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<250"],
  },
};

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";
const apiKey = __ENV.API_KEY || "qk_demo_local_dev_key";

export default function () {
  const unique = `${Date.now()}-${__VU}-${__ITER}`;
  const payload = JSON.stringify({
    targetUrl: `https://example.com/create-benchmark/${unique}`,
  });

  const res = http.post(`${baseUrl}/api/qr/create`, payload, {
    headers: {
      Authorization: `Bearer ${apiKey}`,
      "Content-Type": "application/json",
    },
  });

  check(res, {
    "created": (r) => r.status === 201,
    "has token": (r) => {
      try {
        return r.json("token").length === 12;
      } catch {
        return false;
      }
    },
  });
}
