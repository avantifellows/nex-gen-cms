#!/usr/bin/env python3
"""
Thin wrapper around opendataloader-pdf for use from Go via exec.Command.

Reads a PDF, runs OpenDataLoader's local extraction, and writes the
resulting JSON element array to stdout.

Requirements:
    pip install opendataloader-pdf
    java 11+ in PATH

Usage:
    python scripts/odl_extract.py /absolute/path/to/input.pdf
"""
import sys
import json
import os
import tempfile


def main():
    if len(sys.argv) < 2:
        _fail("usage: odl_extract.py <pdf_path>")

    pdf_path = sys.argv[1]
    if not os.path.isfile(pdf_path):
        _fail(f"file not found: {pdf_path}")

    try:
        import opendataloader_pdf  # noqa: F401
    except ImportError:
        _fail(
            "opendataloader-pdf is not installed. "
            "Run: pip install opendataloader-pdf  (also requires Java 11+)"
        )

    with tempfile.TemporaryDirectory() as tmp_dir:
        try:
            opendataloader_pdf.convert(
                input_path=[pdf_path],
                output_dir=tmp_dir,
                format="json",
            )
        except Exception as exc:
            _fail(f"ODL conversion failed: {exc}")

        # ODL writes <basename>.json into the output directory.
        base = os.path.splitext(os.path.basename(pdf_path))[0]
        json_path = os.path.join(tmp_dir, base + ".json")
        if not os.path.exists(json_path):
            candidates = [f for f in os.listdir(tmp_dir) if f.endswith(".json")]
            if not candidates:
                _fail(
                    "ODL produced no JSON output. "
                    f"Files in tmp dir: {os.listdir(tmp_dir)}"
                )
            json_path = os.path.join(tmp_dir, candidates[0])

        with open(json_path, encoding="utf-8") as fh:
            data = json.load(fh)

    # Write UTF-8 bytes to stdout (Go reads raw bytes). Text-mode stdout on
    # Windows defaults to cp1252 and raises on PDF symbols/private-use chars.
    _write_json_stdout(data)


def _write_json_stdout(obj) -> None:
    payload = json.dumps(obj, ensure_ascii=False).encode("utf-8")
    sys.stdout.buffer.write(payload)
    sys.stdout.buffer.flush()


def _fail(msg: str) -> None:
    _write_json_stdout({"error": msg})
    sys.exit(1)


if __name__ == "__main__":
    main()
