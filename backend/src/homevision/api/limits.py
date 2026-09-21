from fastapi import HTTPException, status
from starlette.responses import JSONResponse
from starlette.types import ASGIApp, Message, Receive, Scope, Send

BODY_TOO_LARGE = "request body exceeds the upload limit"


class UploadLimitMiddleware:
    """Reject request bodies over max_bytes before they are buffered or spooled to disk."""

    def __init__(self, app: ASGIApp, max_bytes: int) -> None:
        self.app = app
        self.max_bytes = max_bytes

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return

        declared = dict(scope["headers"]).get(b"content-length", b"")
        if declared.isdigit() and int(declared) > self.max_bytes:
            response = JSONResponse(
                {"error": BODY_TOO_LARGE}, status_code=status.HTTP_413_CONTENT_TOO_LARGE
            )
            await response(scope, receive, send)
            return

        # Content-Length can be absent (chunked) or wrong, so count what arrives too.
        received = 0

        async def limited_receive() -> Message:
            nonlocal received
            message = await receive()
            if message["type"] == "http.request":
                received += len(message.get("body", b""))
                if received > self.max_bytes:
                    raise HTTPException(status.HTTP_413_CONTENT_TOO_LARGE, BODY_TOO_LARGE)
            return message

        await self.app(scope, limited_receive, send)
