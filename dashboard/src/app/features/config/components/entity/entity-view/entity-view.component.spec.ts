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
    expect(card.textContent).toContain('Filtered by entity');
    expect(card.querySelector('mat-chip')?.textContent).toContain('owner:archive');
    expect(link.getAttribute('href')).toBe('/config/entity/entity-1');
    expect(fixture.nativeElement.querySelector('mat-nav-list')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-chip-listbox')).toBeNull();
  });
});
