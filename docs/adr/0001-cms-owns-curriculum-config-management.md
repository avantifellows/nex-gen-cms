# CMS owns Curriculum Config Management

Curriculum Config Management belongs in Next Generation CMS because it is global content/admin configuration, not an operational LMS workflow. CMS will edit the live `lms_chapter_exam_configs` rows directly through Postgres instead of calling LMS APIs, so the feature can be verified in CMS first and then removed from LMS without leaving CMS coupled to the app it is replacing.
