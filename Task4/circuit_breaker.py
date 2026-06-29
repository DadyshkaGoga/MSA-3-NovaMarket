# Locust scenario for the circuit breaker on /logistics/.
# While "logistics" responds with a timeout (>3s), the first requests wait ~3s and get a fallback
# (NGINX counts them as upstream errors). After 5 consecutive errors the circuit breaker opens for
# 30s: requests stop waiting on the upstream and get an instant fallback -- in locust stats this
# shows up as a sharp drop in response time (from ~3000 ms to a few ms).
from locust import HttpUser, task, constant


class LogisticsUser(HttpUser):
    wait_time = constant(0)

    @task
    def logistics(self):
        with self.client.get(
            "/logistics/track",
            name="/logistics/",
            catch_response=True,
        ) as r:
            # The fallback returns 200 with a "degraded" body -- treat it as the breaker working.
            body = r.text or ""
            if r.status_code == 200 and "degraded" in body:
                r.success()
            elif r.status_code == 200:
                r.success()
            else:
                r.failure(f"unexpected {r.status_code}")
