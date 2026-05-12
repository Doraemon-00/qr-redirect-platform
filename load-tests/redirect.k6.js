import http from "k6/http";
import { check } from "k6";

export const options = {
  scenarios: {
    warm_redirect: {
      executor: "constant-arrival-rate",
      rate: Number(__ENV.RATE || 100),
      timeUnit: "1s",
      duration: __ENV.DURATION || "30s",
      preAllocatedVUs: Number(__ENV.VUS || 50),
      maxVUs: Number(__ENV.MAX_VUS || 200),
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<100"],
  },
};

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";
const token = __ENV.TOKEN;

if (!token) {
  throw new Error("TOKEN env var is required");
}

export default function () {
  const res = http.get(`${baseUrl}/r/${token}`, { redirects: 0 });
  check(res, {
    "redirects with 302": (r) => r.status === 302,
    "has Location header": (r) => Boolean(r.headers.Location),
  });
}
