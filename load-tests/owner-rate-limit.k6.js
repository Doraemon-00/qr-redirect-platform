import http from "k6/http";
import { check } from "k6";

export const options = {
  scenarios: {
    owner_rate_limit: {
      executor: "shared-iterations",
      vus: 10,
      iterations: 80,
      maxDuration: "30s",
    },
  },
};

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";
const apiKey = __ENV.API_KEY || "qk_demo_local_dev_key";

export default function () {
  const res = http.get(`${baseUrl}/api/qr/aaaaaaaaaaaa`, {
    headers: {
      Authorization: `Bearer ${apiKey}`,
    },
  });

  check(res, {
    "normal or rate limited": (r) => r.status === 404 || r.status === 429,
    "rate limit headers exist": (r) => Boolean(r.headers["Ratelimit-Limit"]),
  });
}
