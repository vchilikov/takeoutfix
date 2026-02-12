# TakeoutFix (العربية)

انقل مكتبة Google Photos الخاصة بك بدون فقدان البيانات الوصفية المهمة.
يقوم TakeoutFix بمعالجة ملفات ZIP من Google Takeout واستعادة معلومات مثل تاريخ الالتقاط والموقع والوصف.

[English](../README.md) · [Русский](README.ru.md) · [中文](README.zh-CN.md) · [हिन्दी](README.hi.md) · [Español](README.es.md) · [Français](README.fr.md) · [العربية](README.ar.md) · [Deutsch](README.de.md)

## لماذا TakeoutFix

- يحافظ على بيانات الصور والفيديو الوصفية التي تضيع غالبا أثناء النقل.
- يعمل مباشرة مع ملفات ZIP القياسية من Google Takeout.
- يقدم تدفقا واضحا وموجها داخل الطرفية من البداية للنهاية.
- مناسب للمستخدم العادي بدون الحاجة إلى كتابة سكربتات.

## بدء سريع (3 خطوات)

1. صدّر مكتبة Google Photos من Google Takeout بصيغة ZIP.
2. ضع جميع ملفات `*.zip` في مجلد محلي واحد.
3. افتح الطرفية داخل هذا المجلد وشغّل الأمر الموصى به أدناه.

## التشغيل (موصى به)

شغّل الأمر مباشرة داخل المجلد الذي يحتوي على ملفات ZIP الخاصة بـ Takeout.

macOS/Linux:

```bash
curl -fsSL https://github.com/vchilikov/takeout-fix/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
iwr -useb https://github.com/vchilikov/takeout-fix/releases/latest/download/install.ps1 | iex
```

## التشغيل اليدوي (اختياري)

استخدم هذا الخيار فقط إذا كنت لا تريد استخدام أمر التثبيت السريع.

1. نزّل أحدث ملف تنفيذي لنظامك من [GitHub Releases](https://github.com/vchilikov/takeout-fix/releases).
2. شغّله داخل المجلد الذي يحتوي على ملفات ZIP الخاصة بـ Takeout.

macOS/Linux:

```bash
./takeoutfix
./takeoutfix --workdir /path/to/folder
```

Windows (PowerShell):

```powershell
.\takeoutfix.exe
.\takeoutfix.exe --workdir C:\path\to\folder
```

## ماذا ستحصل عليه

بعد التشغيل الناجح:

- ستكون الملفات المعالجة في `./takeoutfix-extracted/Takeout`.
- سيتم تطبيق البيانات الوصفية على الصور ومقاطع الفيديو المدعومة.
- سيتم كتابة `Tags` من JSON في `Keywords` و`Subject`.
- إذا كان `photoTakenTime.timestamp` مفقودا أو غير صالح، وكان اسم الملف يبدأ بالنمط `YYYY-MM-DD HH.MM.SS`، فسيتم استعادة التاريخ من اسم الملف.
- يمكنك رفع `./takeoutfix-extracted/Takeout` إلى التخزين الجديد.

## المشاكل الشائعة

- `No ZIP archives found in current folder.`
  - يقوم TakeoutFix باكتشاف محتوى Takeout المفكوك مسبقا داخل مجلد العمل تلقائيا.
  - إذا استمرت الرسالة، انقل كل أجزاء ZIP الخاصة بـ Takeout إلى المستوى الأعلى من مجلد العمل، أو شغّل الأداة من مجلد يحتوي مسبقا على محتوى Takeout المفكوك.
- `Corrupt ZIP files found. Processing stopped.`
  - أعد تنزيل الأجزاء التالفة من Google Takeout ثم أعد التشغيل.
- `Missing dependencies: exiftool`
  - استخدم أمر التشغيل الموصى به أعلاه أو ثبّت `exiftool` يدويا.
- `Not enough disk space even with auto-delete enabled.`
  - وفر مساحة إضافية على القرص ثم أعد التشغيل.
- macOS يعرض أن التطبيق غير موثوق
  - أزل العزل ثم أعد التشغيل:
    ```bash
    xattr -d com.apple.quarantine ./takeoutfix
    ```
