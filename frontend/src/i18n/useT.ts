import { useState } from 'react';
import { useTranslation } from 'react-i18next';

export function useT() {
  const { t, i18n } = useTranslation();
  return { t, i18n };
}
