import {enUS, nb} from 'date-fns/locale';

import {LocaleService} from './locale.service';

describe('LocaleService', () => {
  afterEach(() => vi.restoreAllMocks());

  it('keeps Angular locale as a normalized string and exposes the Norwegian date-fns locale', () => {
    vi.spyOn(window.navigator, 'language', 'get').mockReturnValue('nb-NO');
    const service = new LocaleService();

    expect(service.getLocale()).toBe('nb');
    expect(service.getDateLocale()).toBe(nb);
  });

  it('falls back to the English date-fns locale object', () => {
    vi.spyOn(window.navigator, 'language', 'get').mockReturnValue('fr-FR');
    const service = new LocaleService();

    expect(service.getLocale()).toBe('en');
    expect(service.getDateLocale()).toBe(enUS);
  });
});
