import {Label} from '../../../shared/models';
import {resolveLabelLink} from './label-links';

describe('resolveLabelLink', () => {
  it('resolves an encoded HTTP link from a matching label', () => {
    const link = resolveLabelLink({
      organisasjonsnummer: {
        text: 'Brønnøysundregistrene',
        urlTemplate: 'https://virksomhet.brreg.no/nb/oppslag/enheter/{value}',
      },
    }, new Label({key: 'organisasjonsnummer', value: '976 029/100'}));

    expect(link).toEqual({
      text: 'Brønnøysundregistrene',
      href: 'https://virksomhet.brreg.no/nb/oppslag/enheter/976%20029%2F100',
    });
  });

  it.each([
    {text: '', urlTemplate: 'https://example.com/{value}'},
    {text: 'Registry', urlTemplate: 'https://example.com/static'},
    {text: 'Registry', urlTemplate: 'javascript:{value}'},
  ])('rejects invalid link configuration %#', link => {
    expect(resolveLabelLink({owner: link}, new Label({key: 'owner', value: 'archive'}))).toBeNull();
  });
});
