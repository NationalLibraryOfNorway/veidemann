import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';
import {AbilityServiceSignal} from '@casl/angular';

import {AuthService} from '../../../../core';
import {ReportNavigationListComponent} from './report-navigation-list.component';

describe('ReportNavigationListComponent', () => {
  let fixture: ComponentFixture<ReportNavigationListComponent>;
  let readableSubjects: Set<string>;

  beforeEach(async () => {
    readableSubjects = new Set(['pagelog']);

    await TestBed.configureTestingModule({
      imports: [ReportNavigationListComponent],
      providers: [
        provideRouter([]),
        {provide: AuthService, useValue: {isAdmin: () => false}},
        {
          provide: AbilityServiceSignal,
          useValue: {can: (_action: string, subject: string) => readableSubjects.has(subject)},
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ReportNavigationListComponent);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('renders the overview hero and only the authorized report group', () => {
    const page = fixture.nativeElement.querySelector('.destination-page') as HTMLElement;
    const hero = fixture.nativeElement.querySelector('.destination-hero') as HTMLElement;
    const introCard = hero.querySelector('.destination-intro-card') as HTMLElement;
    const artworkCard = hero.querySelector('.destination-artwork-card') as HTMLElement;
    const groupHeadings = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.destination-group h2'))
      .map(heading => heading.textContent.trim());
    const links = fixture.nativeElement.querySelectorAll('.destination-link');

    expect(introCard.querySelector('h1').textContent).toBe('Reports');
    expect(introCard.querySelector('p').textContent.trim())
      .toBe('Explore crawl activity, execution details, and collected logs.');
    expect(getComputedStyle(introCard.querySelector('h1')).marginBottom).toBe('12px');
    expect(hero.firstElementChild).toBe(introCard);
    expect(introCard.nextElementSibling).toBe(artworkCard);
    expect(artworkCard.getAttribute('aria-hidden')).toBe('true');
    expect(artworkCard.classList).toContain('mat-mdc-card-filled');
    expect(artworkCard.classList).not.toContain('mat-mdc-card-outlined');
    for (const heroCard of [introCard, artworkCard]) {
      const style = getComputedStyle(heroCard);
      expect(style.backgroundColor).toBe('rgba(0, 0, 0, 0)');
      expect(style.borderTopWidth).toBe('0px');
      expect(style.boxShadow).toBe('none');
    }
    expect(artworkCard.querySelector('.destination-brand-logo')?.getAttribute('src'))
      .toBe('public/logo/veidemann_logo_inline_black.png');
    expect(artworkCard.querySelector('source')?.getAttribute('srcset'))
      .toBe('public/logo/veidemann_horizontal_white.png');
    expect(hero.querySelector('.destination-assist-chip')).toBeNull();
    expect(groupHeadings).toEqual(['Logs']);
    expect(links.length).toBe(1);
    expect(links[0].getAttribute('href')).toBe('/report/pagelog');
    expect(links[0].textContent).toContain('Page log');
    const avatar = links[0].querySelector('div[mat-card-avatar]') as HTMLElement;
    expect(avatar).not.toBeNull();
    expect(avatar.classList).toContain('config-kind-avatar');
    expect(avatar.querySelector('mat-icon').textContent.trim()).toBe('art_track');
    expect(links[0].querySelector('mat-icon[mat-card-avatar]')).toBeNull();
    expect(getComputedStyle(page).marginTop).toBe('8px');
    expect(getComputedStyle(page).marginRight).toBe('0px');
    expect(getComputedStyle(page).padding).toBe('0px');
    expect(getComputedStyle(hero).paddingInline).toBe('8px');
    expect(getComputedStyle(hero).gap).toBe('8px');
    expect(getComputedStyle(hero).marginBottom).toBe('0');
    expect(getComputedStyle(introCard).minHeight).toBe('320px');
    expect(getComputedStyle(fixture.nativeElement.querySelector('.destination-groups')).paddingBottom)
      .toBe('64px');
  });

  it('groups report destinations without a running-crawls hero shortcut', () => {
    readableSubjects = new Set(['jobexecution', 'crawlexecution', 'pagelog', 'crawllog']);
    fixture.destroy();
    fixture = TestBed.createComponent(ReportNavigationListComponent);
    fixture.detectChanges();

    const groups = Array.from<HTMLElement>(fixture.nativeElement.querySelectorAll('.destination-group'));
    const groupLinks = (group: HTMLElement) => Array.from<HTMLElement>(group.querySelectorAll('mat-card-title'))
      .map(title => title.textContent.trim());

    expect(groups.map(group => group.querySelector('h2').textContent.trim()))
      .toEqual(['Executions', 'Logs']);
    expect(groupLinks(groups[0])).toEqual(['Job execution', 'Crawl execution']);
    expect(groupLinks(groups[1])).toEqual(['Page log', 'Crawl log']);
    expect(fixture.nativeElement.querySelector('.destination-assist-chip')).toBeNull();
  });
});
