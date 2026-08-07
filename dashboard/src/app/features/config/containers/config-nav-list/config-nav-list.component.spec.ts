import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';

import {AuthService} from '../../../../core';
import {Kind} from '../../../../shared/models';
import {ConfigNavListComponent} from './config-nav-list.component';

describe('ConfigNavListComponent', () => {
  let fixture: ComponentFixture<ConfigNavListComponent>;
  let readableSubjects: Set<string>;

  beforeEach(async () => {
    readableSubjects = new Set([Kind[Kind.SEED]]);

    await TestBed.configureTestingModule({
      imports: [ConfigNavListComponent],
      providers: [
        provideRouter([]),
        {provide: AuthService, useValue: {isAdmin: () => false, isCurator: () => false}},
        {
          provide: AbilityServiceSignal,
          useValue: {can: (_action: string, subject: string) => readableSubjects.has(subject)},
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ConfigNavListComponent);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('renders the overview hero and only the authorized configuration group', () => {
    const hero = fixture.nativeElement.querySelector('.destination-hero') as HTMLElement;
    const introCard = hero.querySelector('.destination-intro-card') as HTMLElement;
    const artworkCard = hero.querySelector('.destination-artwork-card') as HTMLElement;
    const groupHeadings = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.destination-group h2'))
      .map(heading => heading.textContent.trim());
    const links = fixture.nativeElement.querySelectorAll('.destination-link');

    expect(introCard.querySelector('h1').textContent).toBe('Configuration');
    expect(introCard.querySelector('p').textContent.trim())
      .toBe('Create and manage crawl targets, jobs, and supporting settings.');
    expect(hero.firstElementChild).toBe(introCard);
    expect(introCard.nextElementSibling).toBe(artworkCard);
    expect(artworkCard.getAttribute('aria-hidden')).toBe('true');
    expect(artworkCard.querySelector('.destination-brand-logo')?.getAttribute('src'))
      .toBe('public/logo/veidemann_logo_inline_black.png');
    expect(artworkCard.querySelector('source')?.getAttribute('srcset'))
      .toBe('public/logo/veidemann_horizontal_white.png');
    expect(groupHeadings).toEqual(['Crawl targets']);
    expect(links.length).toBe(1);
    expect(links[0].getAttribute('href')).toBe('/config/seed');
    expect(links[0].textContent).toContain('Seed');
  });

  it('groups every configuration destination by purpose', () => {
    readableSubjects = new Set([
      Kind.CRAWLENTITY,
      Kind.SEED,
      Kind.CRAWLJOB,
      Kind.CRAWLSCHEDULECONFIG,
      Kind.CRAWLCONFIG,
      Kind.COLLECTION,
      Kind.BROWSERCONFIG,
      Kind.BROWSERSCRIPT,
      Kind.POLITENESSCONFIG,
      Kind.CRAWLHOSTGROUPCONFIG,
      Kind.ROLEMAPPING,
    ].map(kind => Kind[kind]));
    fixture.destroy();
    fixture = TestBed.createComponent(ConfigNavListComponent);
    fixture.detectChanges();

    const groups = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.destination-group'));
    const groupLinks = (group: HTMLElement) => Array.from<HTMLElement>(group.querySelectorAll('mat-card-title'))
      .map(title => title.textContent.trim());

    expect(groups.map(group => group.querySelector('h2').textContent.trim()))
      .toEqual(['Crawl targets', 'Crawl setup', 'Access management']);
    expect(groupLinks(groups[0])).toEqual(['Entity', 'Seed']);
    expect(groupLinks(groups[1])).toEqual([
      'Crawl jobs',
      'Schedule',
      'Crawl config',
      'Collection',
      'Browser config',
      'Browser script',
      'Politeness',
      'Crawl host group',
    ]);
    expect(groupLinks(groups[2])).toEqual(['Users']);
  });
});
