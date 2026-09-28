# Makes the first marks spells leave on the ground of a battle, as png files in
# game/magic/combat/decals. They are made here and drawn by nobody: a first pass, to be replaced
# by pictures that are drawn by hand (replacement folder, spells/<name>/decal_00.png).
#
#   python util/decals/make.py
#
# A mark is as large as a tile of the battlefield is on the screen, 30 by 16 art pixels, or a
# little more. The ground is seen from above at an angle, so a round mark is twice as wide as it
# is high. Its pixels are there or not, there is nothing half see-through: toward its rim a mark
# thins out in a pattern of single pixels, as the pictures of the game do.

import math
import os
import random

from PIL import Image

OUT = os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', '..', 'game', 'magic', 'combat', 'decals')

# colors, darkest first
SCORCH = [(14, 10, 8), (30, 20, 14), (48, 34, 22), (70, 52, 34)]
ASH = [(96, 90, 84), (130, 124, 116)]
FROST = [(236, 248, 255), (190, 226, 250), (140, 190, 236), (96, 150, 210)]
SPARK = [(10, 10, 14), (30, 30, 40), (56, 58, 76)]


def noise(rng, width, height):
    # a value for every pixel that changes slowly from pixel to pixel
    coarse = [[rng.random() for _ in range(width // 3 + 3)] for _ in range(height // 3 + 3)]
    out = [[0.0] * width for _ in range(height)]
    for y in range(height):
        for x in range(width):
            fx, fy = x / 3.0, y / 3.0
            x0, y0 = int(fx), int(fy)
            tx, ty = fx - x0, fy - y0
            top = coarse[y0][x0] * (1 - tx) + coarse[y0][x0 + 1] * tx
            bottom = coarse[y0 + 1][x0] * (1 - tx) + coarse[y0 + 1][x0 + 1] * tx
            out[y][x] = top * (1 - ty) + bottom * ty
    return out


def blotch(rng, width, height, radius_x, radius_y, ramp, specks=None, rough=0.45):
    picture = Image.new('RGBA', (width, height), (0, 0, 0, 0))
    field = noise(rng, width, height)
    middle_x = (width - 1) / 2.0
    middle_y = (height - 1) / 2.0

    for y in range(height):
        for x in range(width):
            # 0 in the middle, 1 at the rim
            far = math.hypot((x - middle_x) / radius_x, (y - middle_y) / radius_y)
            far += (field[y][x] - 0.5) * rough * 2
            if far >= 1:
                continue

            # thinner toward the rim, in a pattern of single pixels
            if far > 0.75 and (x + y) % 2 == 1:
                continue
            if far > 0.9 and (x % 2 == 0 or y % 2 == 0):
                continue

            step = min(len(ramp) - 1, int(far * len(ramp) + (field[y][x] - 0.5)))
            picture.putpixel((x, y), ramp[max(0, step)] + (255,))

    if specks:
        for _ in range(int(radius_x * radius_y / 6)):
            angle = rng.random() * 2 * math.pi
            far = rng.random() ** 0.5 * 0.85
            x = int(round(middle_x + math.cos(angle) * far * radius_x))
            y = int(round(middle_y + math.sin(angle) * far * radius_y))
            if 0 <= x < width and 0 <= y < height and picture.getpixel((x, y))[3]:
                picture.putpixel((x, y), rng.choice(specks) + (255,))

    return picture


def cracks(rng, width, height, arms, reach_x, reach_y, ramp):
    # a small burn with lines that run out of it
    picture = blotch(rng, width, height, reach_x * 0.4, reach_y * 0.4, ramp, rough=0.3)
    middle_x = (width - 1) / 2.0
    middle_y = (height - 1) / 2.0

    for arm in range(arms):
        angle = (arm + rng.random() * 0.6) / arms * 2 * math.pi
        length = 0.55 + rng.random() * 0.45
        steps = int(max(reach_x, reach_y) * 2)
        for step in range(steps):
            part = step / float(steps)
            if part > length:
                break
            angle += (rng.random() - 0.5) * 0.35
            x = int(round(middle_x + math.cos(angle) * part * reach_x))
            y = int(round(middle_y + math.sin(angle) * part * reach_y))
            if 0 <= x < width and 0 <= y < height:
                shade = ramp[min(len(ramp) - 1, int(part * len(ramp)))]
                picture.putpixel((x, y), shade + (255,))

    return picture


def main():
    os.makedirs(OUT, exist_ok=True)

    for number in range(3):
        rng = random.Random(1000 + number)
        blotch(rng, 34, 18, 13 + number, 6.5 + number * 0.5, SCORCH, specks=ASH).save(os.path.join(OUT, 'scorch_%d.png' % number))

    for number in range(3):
        rng = random.Random(2000 + number)
        blotch(rng, 46, 24, 19 + number, 9.5 + number * 0.5, SCORCH, specks=ASH, rough=0.55).save(os.path.join(OUT, 'crater_%d.png' % number))

    for number in range(3):
        rng = random.Random(3000 + number)
        blotch(rng, 34, 18, 13 + number, 6.5 + number * 0.5, FROST, specks=[FROST[0]], rough=0.6).save(os.path.join(OUT, 'frost_%d.png' % number))

    for number in range(3):
        rng = random.Random(4000 + number)
        cracks(rng, 34, 18, 6 + number, 15, 7.5, SPARK).save(os.path.join(OUT, 'spark_%d.png' % number))

    print('written to', os.path.normpath(OUT))


main()
