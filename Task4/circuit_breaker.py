# Locust scenario for the circuit breaker on /logistics/. The upstream times out (>3s); after 5
# errors the breaker opens and fallbacks become instant -- response time drops from ~3s to a few ms.
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
            # The fallback returns 200 with a "degraded" body -- treat any 200 as the breaker working.
            if r.status_code == 200:
                r.success()
            else:
                r.failure(f"unexpected {r.status_code}")
