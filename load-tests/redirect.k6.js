import http from "k6/http";
import { check } from "k6";

export const options = {
  scenarios: {
    redirect_smoke: {
      executor: "constant-vus",
      vus: 10,
      duration: "30s",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<100"],
  },
};

const baseUrl = __ENV.BASE_URL || "http://localhost:8080";
const token = __ENV.TOKEN || "replace-with-token";

export default function () {
  const res = http.get(`${baseUrl}/r/${token}`, { redirects: 0 });
  check(res, {
    "redirect or known placeholder": (r) => r.status === 302 || r.status === 501,
  });
}
