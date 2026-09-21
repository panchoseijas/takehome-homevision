import argparse
import logging

import uvicorn

from homevision.api.app import Config, create_app
from homevision.vision import Detector, Params

KEEP_ALIVE_TIMEOUT_SECONDS = 60
SHUTDOWN_TIMEOUT_SECONDS = 10


def main() -> None:
    parser = argparse.ArgumentParser(description="Serve the checkbox detection API.")
    parser.add_argument("--host", default="0.0.0.0", help="address to listen on")
    parser.add_argument("--port", type=int, default=8080, help="TCP port to listen on")
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")

    params = Params()
    app = create_app(Detector(params), Config(max_pixels=params.max_pixels))
    # uvicorn drains in-flight requests on SIGINT/SIGTERM before exiting.
    uvicorn.run(
        app,
        host=args.host,
        port=args.port,
        timeout_keep_alive=KEEP_ALIVE_TIMEOUT_SECONDS,
        timeout_graceful_shutdown=SHUTDOWN_TIMEOUT_SECONDS,
        log_config=None,
    )


if __name__ == "__main__":
    main()
