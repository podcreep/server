
ALTER TABLE podcasts
  ADD COLUMN is_image_custom NUMBER;

UPDATE podcasts SET is_image_custom = 0;
