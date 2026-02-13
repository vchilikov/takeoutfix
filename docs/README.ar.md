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
- إذا كان طابع وقت الالتقاط في JSON مفقودا أو غير صالح وكان اسم الملف يبدأ بالنمط `YYYY-MM-DD HH.MM.SS`، فسيتم استعادة التاريخ من اسم الملف.
- يتم حفظ تقرير تشغيل مفصل في `./.takeoutfix/reports/report-YYYYMMDD-HHMMSS.json`.
- يمكنك رفع `./takeoutfix-extracted/Takeout` إلى التخزين الجديد.

## المشاكل الشائعة

- `No ZIP files or extracted Takeout data found in this folder.`
  - انقل كل أجزاء ZIP الخاصة بـ Takeout إلى المستوى الأعلى من مجلد العمل، أو شغّل الأداة من مجلد يحتوي مسبقا على محتوى Takeout المفكوك.
- `Some ZIP files are corrupted. Please re-download them and run again.`
  - أعد تنزيل الأجزاء التالفة من Google Takeout ثم أعد التشغيل.
- `Step 1/3: Checking dependencies... missing`
  - استخدم أمر التشغيل الموصى به أعلاه أو ثبّت `exiftool` يدويا.
- `Not enough free disk space to continue.`
  - وفر مساحة إضافية على القرص ثم أعد التشغيل.
- macOS يعرض أن التطبيق غير موثوق
  - أزل العزل ثم أعد التشغيل:
    ```bash
    xattr -d com.apple.quarantine ./takeoutfix
    ```
