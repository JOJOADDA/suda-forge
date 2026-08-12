# SUDA FORGE production delivery layer

This directory contains the first production deployment layer for SUDA FORGE. It targets an Ubuntu or Debian server with systemd, PostgreSQL, Go, Node.js, pnpm, Caddy, and LXC available or installable by the operator.

## Install

Run from a checked-out repository:

```bash
sudo SUDA_INSTALL_DIR=/opt/suda-forge bash infra/install.sh suda.example.com
```

For a controlled bootstrap where public DNS is not ready yet, use `SUDA_SKIP_DNS_CHECK=1`. Do not expose the application publicly until DNS and HTTPS are configured.

The installer creates `/etc/suda-forge/suda-forge.env` from the example file on first run. Replace the database URL and optional runtime URLs before production use. Migrations are applied through `scripts/migrate.sh`, the frontend is built into `apps/web/dist`, the backend is built into `bin/suda-forge`, and a systemd unit is installed at `/etc/systemd/system/suda-forge.service`.

## Operations

```bash
systemctl status suda-forge
journalctl -u suda-forge -f
systemctl restart suda-forge
bash infra/lib/health-check.sh
```

Caddy serves the built frontend and proxies `/api/*`, `/healthz`, `/health`, `/ready`, and `/events` to the loopback Go server. The current Caddy template is intentionally conservative: project preview routing, authentication, and domain/certificate lifecycle still require the next production phase.

## Important boundary

The service currently runs as root because SUDA FORGE's LXC runtime provider has not yet been converted to a delegated non-root service account. This is a temporary operational requirement and must be replaced before a security-sensitive public deployment.


## Automated deployment

`infra/deploy.sh` يبني release artifact من working tree نظيف، يشغّل اختبارات Go وفحص TypeScript وبناء Vite، ثم يرفع artifact إلى خادم خارجي عبر SSH. على الخادم، ينفذ `infra/remote-activate.sh` التحقق من checksum، تطبيق migrations، إنشاء release مستقل، تبديل symlink باسم `current`، إعادة تشغيل systemd، تحديث Caddy، ثم health gate.

يتطلب deploy الآلي أن يكون المستخدم المحلي قادرًا على الاتصال بالخادم دون تفاعل عبر SSH، وأن يملك المستخدم البعيد صلاحية `sudo -n` لتشغيل سكربت التفعيل. يجب إنشاء `/etc/suda-forge/suda-forge.env` على الخادم مسبقًا باستخدام `infra/install.sh` ومراجعة `DATABASE_URL` والأسرار يدويًا.

مثال نشر كامل:

```bash
infra/deploy.sh \
  --host server.example.com \
  --user ubuntu \
  --hostname suda.example.com
```

فحص البناء والتغليف دون رفع أو تغيير الخادم:

```bash
infra/deploy.sh \
  --host server.example.com \
  --dry-run
```

إذا كان Go أو Node أو npm registry غير متاح في بيئة البناء، يمكن تجاوز الاختبارات فقط عند وجود سبب تشغيلي واضح:

```bash
SUDA_SKIP_TESTS=1 infra/deploy.sh --host server.example.com
```

لا يطبق `SUDA_SKIP_TESTS=1` migrations أو يتجاوز health gate؛ فهو يتجاوز اختبار Go وtypecheck فقط مع استمرار بناء frontend. ولتجاوز migrations عمدًا استخدم `--skip-migrations` بعد التأكد من تطبيقها يدويًا.

يحتفظ النظام بآخر releases تحت `/opt/suda-forge-releases` ويحتفظ افتراضيًا بثلاثة إصدارات. إذا فشل تشغيل systemd أو Caddy أو health check، يعيد `remote-activate.sh` symlink إلى الإصدار السابق ويعيد تشغيل الخدمة. migrations قاعدة البيانات forward-only؛ لا يتم التراجع عنها تلقائيًا، ولذلك يجب أخذ backup قبل نشر migration حساسة.

## Server prerequisites

يجب أن يتوفر على الخادم Ubuntu أو Debian، systemd، SSH، sudo، PostgreSQL، Go، Node.js، pnpm، curl، tar، sha256sum، وCaddy عند الحاجة إلى HTTPS. كما يجب أن يكون runtime الخاص بـ LXC مجهزًا مسبقًا إذا كانت Project Computers مطلوبة. التشغيل الحالي للخدمة يستخدم root بسبب حدود LXC، ولا ينبغي فتح الخدمة للعامة قبل إكمال فصل صلاحيات runtime والمصادقة والصلاحيات.


## Health gate and rollback

كل release جديد يمر عبر health gate بعد إعادة تشغيل systemd وتحديث Caddy. إذا فشل تشغيل الخدمة أو فشل فحص `/healthz` أو أصبح إعداد Caddy غير صالح، يعيد التفعيل `current` إلى الإصدار السابق، يعيد تثبيت unit السابق، ويستعيد Caddyfile السابق عند توفره.

يمكن اختبار مسار التفعيل محليًا دون خادم حقيقي:

```bash
infra/tests/deployment-logic-test.sh
infra/tests/production-delivery-test.sh
```

يختبر المسار الأول تفعيل release ناجحًا ثم يحاكي فشل health gate ويتأكد من عودة symlink إلى الإصدار السابق. أما `--dry-run` في `infra/deploy.sh` فيبني artifact ويحسب checksum دون رفع أو تنفيذ أي تغيير خارجي:

```bash
infra/deploy.sh --dry-run
```

يتطلب النشر الحقيقي أن يكون working tree نظيفًا، وأن تكون جلسة SSH غير تفاعلية، وأن يملك المستخدم البعيد `sudo -n`. لا يتم rollback تلقائي لعمليات PostgreSQL؛ فهي forward-only ويجب أخذ backup قبل نشر migrations حساسة.


## Network and DNS check before GitHub push

يمكن فحص المسار الافتراضي وDNS وHTTPS وGitHub قبل الرفع باستخدام:

```bash
infra/check-network-and-push.sh --attempts 3 --wait 3
```

للمحاولة الآلية لإصلاح DNS، يستخدم السكربت `resolvectl` أولًا ويطبق DNS runtime على واجهة الشبكة الافتراضية:

```bash
sudo infra/check-network-and-push.sh --fix-dns
```

إذا لم يتوفر `resolvectl`، يرفض السكربت تعديل `/etc/resolv.conf` افتراضيًا. يمكن السماح بتعديل مؤقت مع أخذ نسخة احتياطية واستعادتها عند انتهاء السكربت:

```bash
sudo infra/check-network-and-push.sh --fix-dns --persist-resolv-conf
```

ولتنفيذ الرفع فقط بعد نجاح جميع فحوصات الاتصال:

```bash
infra/check-network-and-push.sh --fix-dns --push
```

السكربت لا يتجاوز أخطاء DNS أو HTTPS ولا ينفذ `git push` عند فشل الفحوصات. كما أن DNS قد يكون محجوبًا من بيئة التشغيل نفسها؛ في هذه الحالة لا يستطيع أي سكربت محلي إصلاح الحجب الخارجي، ويجب تغيير الشبكة أو إعدادات الـ runner أو تنفيذ الأمر من خادم يملك اتصالًا خارجيًا.
