import asyncio
import os

import asyncpg
import requests

API_URL = os.environ.get("API_URL", "https://spider.omniplex.gg")


async def main():
    conn = await asyncpg.connect(os.environ["DATABASE_URL"])

    uids = await conn.fetch('SELECT user_id FROM users')

    for uid in uids:
        user_id = uid['user_id']
        print(f"Updating CDN for user {user_id}...")

        await conn.execute('DELETE FROM internal_user_cache__discord WHERE id = $1', user_id)

        resp = requests.get(f"{API_URL}/platform/user/{user_id}?platform=discord")

        if not resp.ok:
            print(f"Failed to update cache for user {user_id}")

        await asyncio.sleep(0.5)

    await conn.close()

asyncio.run(main())
