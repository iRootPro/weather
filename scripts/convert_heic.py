#!/usr/bin/env python3
"""
Конвертер HEIC в JPEG используя pillow-heif
Использование: python3 convert_heic.py <input.heic> <output.jpg>
"""
import sys
from pillow_heif import register_heif_opener
from PIL import Image

# Регистрируем HEIF opener для PIL
register_heif_opener()

if len(sys.argv) != 3:
    print("Usage: convert_heic.py <input.heic> <output.jpg>", file=sys.stderr)
    sys.exit(1)

input_path = sys.argv[1]
output_path = sys.argv[2]

try:
    # Открываем HEIC файл
    image = Image.open(input_path)

    # Сохраняем как JPEG с высоким качеством
    image.save(output_path, "JPEG", quality=90, optimize=True)

    print(f"Successfully converted {input_path} to {output_path}")
    sys.exit(0)
except Exception as e:
    print(f"Error converting image: {e}", file=sys.stderr)
    sys.exit(1)
