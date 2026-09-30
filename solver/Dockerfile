FROM python:3.12.12-slim

WORKDIR /app
COPY pyproject.toml ./
RUN pip install --no-cache-dir ".[test]"
COPY app ./app
COPY tests ./tests

EXPOSE 8090
CMD ["python", "-m", "app.server"]
