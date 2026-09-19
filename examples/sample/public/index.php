<?php

header('Content-Type: text/html; charset=utf-8');

$dbStatus = '未接続';
$dbError = null;

$dbHost = getenv('DB_HOST') ?: 'db';
$dbPort = getenv('DB_PORT') ?: '3306';
$dbName = getenv('DB_DATABASE') ?: 'sample';
$dbUser = getenv('DB_USERNAME') ?: 'sample_user';
$dbPass = getenv('DB_PASSWORD') ?: 'sample_pass';

try {
    $pdo = new PDO("mysql:host={$dbHost};port={$dbPort};dbname={$dbName}", $dbUser, $dbPass, [
        PDO::ATTR_TIMEOUT => 2,
        PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
    ]);
    $dbStatus = '接続成功 (MySQL OK)';
} catch (Exception $e) {
    $dbStatus = '接続エラー';
    $dbError = $e->getMessage();
}

?>
<!DOCTYPE html>
<html lang="ja">
<head>
    <meta charset="UTF-8">
    <title>devctl Sample Application</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0f172a; color: #f8fafc; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; }
        .card { background: #1e293b; border-radius: 12px; padding: 2.5rem; max-width: 600px; width: 90%; box-shadow: 0 10px 25px rgba(0,0,0,0.5); }
        h1 { margin-top: 0; color: #38bdf8; font-size: 1.75rem; }
        .status { padding: 0.75rem 1rem; border-radius: 6px; margin: 1rem 0; font-weight: bold; }
        .ok { background: #064e3b; color: #6ee7b7; border: 1px solid #059669; }
        .err { background: #450a0a; color: #fca5a5; border: 1px solid #dc2626; }
        ul { list-style: none; padding: 0; }
        li { padding: 0.5rem 0; border-bottom: 1px solid #334155; display: flex; justify-content: space-between; }
        .label { color: #94a3b8; }
        .val { font-family: monospace; }
    </style>
</head>
<body>
    <div class="card">
        <h1>devctl Sample Application</h1>
        <p>devctl 管理下のサンプルコンテナ環境が正常に稼働しています。</p>

        <div class="status <?= $dbError ? 'err' : 'ok' ?>">
            データベース: <?= htmlspecialchars($dbStatus) ?>
            <?php if ($dbError): ?>
                <div style="font-size: 0.85rem; font-weight: normal; margin-top: 0.5rem;"><?= htmlspecialchars($dbError) ?></div>
            <?php endif; ?>
        </div>

        <ul>
            <li><span class="label">Host</span><span class="val"><?= htmlspecialchars($_SERVER['HTTP_HOST'] ?? 'localhost') ?></span></li>
            <li><span class="label">PHP Version</span><span class="val"><?= PHP_VERSION ?></span></li>
            <li><span class="label">App IP</span><span class="val"><?= htmlspecialchars($_SERVER['SERVER_ADDR'] ?? '10.11.0.2') ?></span></li>
            <li><span class="label">DB Host</span><span class="val"><?= htmlspecialchars($dbHost) ?></span></li>
            <li><span class="label">DB IP</span><span class="val"><?= htmlspecialchars(getenv('DB_IP') ?: '10.11.0.3') ?></span></li>
        </ul>
    </div>
</body>
</html>
