import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';

import {ConfigObject, Kind, Label, Meta} from '../../../../../shared/models';
import {EntityViewComponent} from './entity-view.component';

describe('EntityViewComponent', () => {
  let fixture: ComponentFixture<EntityViewComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [EntityViewComponent],
      providers: [provideRouter([])],
    }).compileComponents();

    fixture = TestBed.createComponent(EntityViewComponent);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'entity-1',
      kind: Kind.CRAWLENTITY,
      meta: new Meta({
        name: 'Example entity',
        labelList: [new Label({key: 'owner', value: 'archive'})],
      }),
    }));
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('renders entity context as an outlined card with labels and a details link', () => {
    const card = fixture.nativeElement.querySelector('mat-card') as HTMLElement;
    const link = fixture.nativeElement.querySelector('a') as HTMLAnchorElement;

    expect(card.classList).toContain('mat-mdc-card-outlined');
    expect(card.textContent).toContain('Example entity');
    expect(card.querySelector('mat-chip')?.textContent).toContain('owner:archive');
    expect(link.getAttribute('href')).toBe('/config/entity/entity-1');
    expect(fixture.nativeElement.querySelector('mat-nav-list')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-chip-listbox')).toBeNull();
  });

  it('renders description and created/modified metadata when available', async () => {
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'entity-2',
      kind: Kind.CRAWLENTITY,
      meta: new Meta({
        name: 'Documented entity',
        description: 'Seeds related to a specific archiving project.',
        created: '2024-01-15T10:00:00Z',
        createdBy: 'andreas.borsheim@nb.no',
        lastModified: '2024-06-01T08:30:00Z',
        lastModifiedBy: 'andreas.borsheim@nb.no',
      }),
    }));
    fixture.detectChanges();
    await fixture.whenStable();

    const card = fixture.nativeElement.querySelector('mat-card') as HTMLElement;

    expect(card.textContent).toContain('Seeds related to a specific archiving project.');
    expect(card.textContent).toContain('Created');
    expect(card.textContent).toContain('Created by');
    expect(card.textContent).toContain('andreas.borsheim@nb.no');
    expect(card.textContent).toContain('Last modified');
    expect(card.textContent).toContain('Last modified by');
  });

  it('omits the content section entirely when there is no extra metadata', () => {
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'entity-3',
      kind: Kind.CRAWLENTITY,
      meta: new Meta({name: 'Bare entity'}),
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('mat-card-content')).toBeNull();
  });
});
