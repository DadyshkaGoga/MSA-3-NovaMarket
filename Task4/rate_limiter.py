# Locust scenario for the rate limiter.
# Two user classes with different channels (X-Client-Type header):
#   web    -> 50 r/s per IP,
#   mobile -> 30 r/s per IP.
# Over the limit NGINX returns 429 -- in locust stats this shows up as failures on /api/.
from locust import HttpUser, task, constant


class WebUser(HttpUser):
    # No pause between requests, to quickly exceed the channel limit.
    wait_time = constant(0)

    @task
    def api(self):
        with self.client.get(
            "/api/resource",
            headers={"X-Client-Type": "web"},
            name="/api/ [web]",
            catch_response=True,
        ) as r:
            if r.status_code == 429:
                r.failure("429 rate limited (web > 50 r/s)")


class MobileUser(HttpUser):
    wait_time = constant(0)

    @task
    def api(self):
        with self.client.get(
            "/api/resource",
            headers={"X-Client-Type": "mobile"},
            name="/api/ [mobile]",
            catch_response=True,
        ) as r:
            if r.status_code == 429:
                r.failure("429 rate limited (mobile > 30 r/s)")
