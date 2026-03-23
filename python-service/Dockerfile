FROM python:3.12-slim

WORKDIR /app

COPY pyproject.toml .
RUN pip install --no-cache-dir .

COPY gen/ /app/gen/
COPY src/ /app/src/

WORKDIR /app/src
CMD ["python", "main.py"]
