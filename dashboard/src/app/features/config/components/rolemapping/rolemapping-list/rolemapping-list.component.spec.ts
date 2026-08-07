import {DestroyRef} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {of} from 'rxjs';

import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {ConfigObject, Kind, ListDataSource, Role, RoleMapping} from '../../../../../shared/models';
import {RoleMappingListComponent} from './rolemapping-list.component';

describe('RoleMappingListComponent', () => {
  let fixture: ComponentFixture<RoleMappingListComponent>;
  let component: RoleMappingListComponent;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [RoleMappingListComponent],
      providers: [...provideCoreTesting],
    }).compileComponents();

    fixture = TestBed.createComponent(RoleMappingListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('keeps its separate email/group renderer and configuration selection behavior', () => {
    const row = new ConfigObject({
      id: 'mapping-1',
      kind: Kind.ROLEMAPPING,
      roleMapping: new RoleMapping({email: 'curator@example.com', roleList: [Role.CURATOR]}),
    });
    component.dataSource = ListDataSource.fromQuery({
      query$: of('query'),
      load: () => of(row),
      destroyRef: fixture.componentRef.injector.get(DestroyRef),
    });
    const selections: ConfigObject[][] = [];
    component.selectedChange.subscribe(value => selections.push(value));
    fixture.detectChanges();

    const item = fixture.nativeElement.querySelector('.item-row') as HTMLElement;
    expect(item.textContent).toContain('curator@example.com');
    expect(item.textContent).toContain('CURATOR');
    expect(item.querySelector('mat-chip-set')).toBeNull();
    expect(fixture.nativeElement.querySelector('.master-selection-control')).toBeNull();

    component.onCheckboxToggle(row);
    fixture.detectChanges();
    expect(selections).toEqual([[row]]);
    expect(item.classList).toContain('row-checked');
  });
});
